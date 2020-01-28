# crawlr

crawlr recursively crawls URLs.

## Usage

### Build & run

Make sure that your local 8000 and 5432 ports aren't allocated by any local
services before running the following command, otherwise the services will fail
to start. This following command will spin up a postgres container, a container
running the crawlr API (on localhost:8000/), and a worker container for the
crawler.

```bash
DSN="postgresql://user:test@db:5432/crawlr" MAX_WORKERS=30 docker-compose -f ./deployments/docker-compose.yml up --build

```

Environment variable meanings:

- `DSN`: connection string for the API server and crawler to connect to the
  database
- `MAX_WORKERS`: specifies the number of workers that the crawler can spin up at
  a time

To destroy and recreate the database/clean up:

```bash
docker container prune && docker volume prune
```

To run all tests:

```bash
go test -v ./...
```

## API

### `POST /crawl`

**Request**

```json
{
  "url": "somehost.com",
  "levels": 1
}
```

- url `string`: Represents the URL to crawl.
- levels `int`: Represents the number of levels of recursion.

**Response**

Returns the request body

**Example**

To create a CrawlRequest starting from google.com with 2 levels of recursion:

```bash
curl localhost:8000/crawl --data '{"url": "google.com", "levels": 2}'
```

### `GET /status/:id`

**Response**

Returns the current status of the CrawlRequest.

**Example**

To check the status of the CrawlRequest with id `1`:

```bash
curl localhost:8000/status/1
```

### `GET /results/:id`

**Response**

Returns a JSON object containing counts of each unique host name found while
crawling (excluding counts of the original hostname in the supplied URL when the
CrawlRequest was created).

**Example**

To check the results of the CrawlRequest with id `1`:

```bash
curl localhost:8000/results/1
```

## Design Decisions

My design is based on the idea that the internet can be represented as a graph,
where pages are nodes and a link on a page represents a directed edge from one
page to another (for example, if `https://google.com` had a link to
`https://maps.google.com` in its html body, I would create a directed edge from
the page node `https://google.com` to the page node `https://maps.google.com`).

I cut down on the links I crawled by ignoring any urls with non http/https
protocols, as well as stripping all urls of their fragments. However, I
considered urls with different protocols to be different pages, even if they
otherwise had the same host name, path, and query. (Ex: `https://google.com` was
treated as a different page than `http://google.com`).

Based on the requirements given, I optimized for scalability at the expense of
adding a little additional complexity into the system. There are 3 overall parts
to this system: the API server, the crawler, and the database. The API server
and crawler each live in their own container and are stateless, so they can be
scaled up or down as needed.

### API Server

When a request to the `/crawl` endpoint is made, the API server creates a
CrawlRequest and inserts the relevant information into the `crawl_requests`
table. The server also creates a task representing the first page node to be
crawled (containing the id of the associated CrawlRequest, the page URL to be
crawled, the current level of recursion (0), and the status of the task
`NOT_STARTED`) and stores that in the `task` table in the database.

When a request to the `/status/:id` endpoint is made, the API server retrieves
all tasks associated with the CrawlRequest id up to the second to last level and
counts how many tasks are `COMPLETED`,`FAILED`, or `IN_PROGRESS` and returns
these metrics.

When a request to the `/results/:id` endpoint is made, the API server retrieves
all tasks associated with the CrawlRequest id and counts the unique hosts seen.
This works because the tasks represent all the pages that were crawled through
the graph for a single CrawlRequest.

### Crawler

The crawler continually retrieves more incomplete tasks from the `tasks` table.
For every task, the crawler will check if a page node has already been created
for the page URL in the `page_nodes` table. 

If no such page node exists, the worker will create a page and insert it into
the `page_nodes` table and make a GET request to the URL. It will then create
new nodes for all the new hosts found, as well as edges from the current page
nodes to the newly created page nodes. Otherwise, if a page has already been
created for the URL, the `task` will retrieve all page nodes with edges where
the source node is the current page node. 

Then the crawler determines based on the current level and the total number of
expected levels for the CrawlRequest whether to create more tasks. If more tasks
should be created, the crawler uses the retrieved URLs of pages (from found
edges) or the retrieved URLs from the GET request, and inserts new tasks into
the database, incrementing the level of recursion by 1. Once all new tasks (if
any) are inserted into the database, the worker marks the current task as
`COMPLETED` and finishes its job.

## Deployment to Production

Since everything is containerized, it should be relatively simple to deploy this
to whatever container orchestration system is preferred. We could use a CI/CD
tool like CircleCI or Github Actions to run unit and integration tests every
time a new feature branch is merged into master, and to do automatic deploys to
a staging or testing environment where an engineer could do additional QA or
testing if necessary. Finally, the engineer can manually deploy changes to a
production environment after the new changes have been tested to their
satisfaction.

We also might want to use a hosted solution for the database instead of
maintaining our own, such as AWS RDS or Google Cloud SQL. (I also barely set up
the database in this solution, so I'd change the Dockerfile to pass in the
database username and password as build arguments instead of just specifying it
in the file itself for security reasons).

In order to scale this, we'd need to put a load balancer in front of the API
server to handle requests (and probably need to set up a domain for our API
server, as well as provisioning TLS/SSL certs if we want our API to use HTTPS),
but many container orchestration systems (like Kubernetes) make this easy.

## Future Work

For simplicity's sake, I built this system with the invariant that once a page
has been crawled, it'll never be updated again. This means that even if a page
has changed since the last time we crawled it, the system doesn't make any
updates to the nodes in the graph if we've already crawled that page. This
increases the speed of individual crawl requests, but decreases the accuracy of
returned results. Future work could include checking Last-Modified, Expires,
Cache-Content, or Etag headers for a page to determine whether we should update
the page node and its edges (or keeping a timestamp for when we last crawled a
page node, and always recrawling a page node after a certain amount of time has
passed). This would make the results returned more accurate.

I don't have graceful server shutdowns or worker retries implemented, so if a
worker dies in the middle of completing a task, the CrawlRequest related to the
task will never be completed. I could create an additional service or function
to find tasks that have been `IN_PROGRESS` for more than a specified amount of
time, and restart that task by setting the task status back to `NOT_STARTED`.

Other interesting things I'd like to add: distributed request tracing throughout
the API and tasks, implementing better logging for the crawler/API (more
informative JSON error messages), adding metrics for long it takes to crawl an
average page, etc. I'd also like to add more thorough test coverage. Currently
I'm only testing http response body html parsing and URL filtering functionality
since I thought it would take up too much time to figure out how to mock the
database client for unit testing functions that involve database access.

In the future, to make this a more polite web crawler, it'd be neat to add ways
so the web crawler could rate-limit itself or to follow the robots.txt/metadata
to only crawl content that the site owner wants to be crawled.

## Thoughts on Scalability

The api and crawler should both be easily horizontally scalable, as they are
stateless and rely on the database to store state. The crawler is also
vertically scalable, as we can scale up the number of workers per crawler via a
command line flag. I'm slightly worried about how to make the database more
scalable - I don't have as much experience there, but I'd love to learn more
about it.

## Learnings

My original design involved doing all of the crawling for a single CrawlRequest
on a single crawler, which involved working with a lot of goroutines, wait
groups, and mutexes. The results/status would then be stored in the database
when the CrawlRequest was complete. I redesigned the system to distribute tasks
among crawlers instead because some pages have hundreds and thousands of links
while others only have 1 or 2 links. This made it hard to reason about the work
that each crawler would do in my first design - even if we scaled up the number
of workers we had, some crawlers might end up with many magnitudes more work
than other crawlers, based on the URL that they're responsible for crawling.

When implementing the distributed task system, I ran into some issues working
with concurrent database reads/writes (had deadlock situations when attempting
to update edges for a page node, which involved also creating new page nodes). I
simplified my database queries which seemed to help (I ran some manual crawl
request tests to check for deadlocks or other errors and didn't see any), but
I'd love to learn more about how to improve my system/algorithm design in a way
that I can provably avoid concurrency issues.
