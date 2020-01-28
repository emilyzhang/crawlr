package graphcrawler

import (
	"fmt"
	"sync"
	"time"

	"github.com/emilyzhang/crawlr/crawlerdb"
)

// GraphCrawler represents a server containing maxWorkers number of workers.
type GraphCrawler struct {
	maxWorkers int
	db         *crawlerdb.Postgres
	wg         *sync.WaitGroup

	// TODO: add proper logging
}

// New creates a new GraphCrawler.
func New(dbDSN string, maxWorkers int) (*GraphCrawler, error) {
	// tries connecting to the database 3 times until it gives up
	retries, count, sleep := 3, 0, 5
	db, err := crawlerdb.New(dbDSN)
	for err != nil {
		if count > retries {
			return nil, err
		}
		time.Sleep(time.Duration(sleep) * time.Second)
		sleep += 3
		db, err = crawlerdb.New(dbDSN)
		count++
	}

	return &GraphCrawler{
		db:         db,
		maxWorkers: maxWorkers,
		wg:         &sync.WaitGroup{},
	}, nil
}

// Start starts the GraphCrawler server, which will spawn up to maxWorkers
// number of workers at a time to grab tasks from the database and complete
// them.
func (c *GraphCrawler) Start() {
	fmt.Println("Starting graph crawler. Hello world!")
	for {
		for i := 0; i < c.maxWorkers; i++ {
			t, err := c.db.FindIncompleteTask()
			if err != nil {
				// don't want to log error if there are simply no tasks yet
				if err != crawlerdb.ErrNoTasksAvailable {
					c.handleError(t, err)
				}
			} else {
				c.wg.Add(1)
				go c.run(t)
			}
		}
		c.wg.Wait()
	}
	// TODO: implement graceful server shut down

	// TODO: implement task retries (when the server is unexpectedly shut down,
	// some tasks may be left in "IN_PROGRESS" state after worker death)
}

// run completes a task by grabbing the page associated with the task, finding
// the next pages for this page, and adding new tasks for those pages.
func (c *GraphCrawler) run(t *crawlerdb.Task) {
	defer c.wg.Done()
	fmt.Printf("CrawlRequest %d: Starting new task (url %s, level %d)\n", t.CrawlRequestID, t.PageURL, t.CurrentLevel)

	cr, err := c.db.GetCrawlRequest(t.CrawlRequestID)
	if err != nil {
		c.handleError(t, err)
		return
	}
	// return immediately if we're on the last level of recursion, we don't need to crawl
	// those hosts, we just needed the task to be created for host counting purposes
	if t.CurrentLevel == cr.Levels {
		c.db.UpdateTaskStatus(t.ID, "COMPLETED")
		return
	}

	// get relevant page node
	p, err := c.db.UpsertPage(t.PageURL)
	if err != nil {
		c.handleError(t, err)
		return
	}
	page, err := c.db.GetPage(p)
	if err != nil {
		c.handleError(t, err)
		return
	}

	// find next urls, either using the already unfolded graph, or by crawling
	// the current page
	var urls []string
	if !page.CrawledStatus {
		urls, err = c.crawlPage(page)
		if err != nil {
			c.handleError(t, err)
			return
		}
	} else {
		urls, err = c.nextPagesFromEdges(page)
		if err != nil {
			c.handleError(t, err)
			return
		}
	}

	// add tasks for outlinks on the page
	err = c.addNewTasks(t, cr.Levels, urls)
	if err != nil {
		c.handleError(t, err)
		return
	}

	// updates task status
	err = c.db.UpdateTaskStatus(t.ID, "COMPLETED")
	if err != nil {
		c.handleError(t, err)
		return
	}
}

// crawlPage unfolds the graph by crawling the page and adding new page nodes
// and edges associated with the current page and returns a slice of strings
// representing urls for the next pages.
func (c *GraphCrawler) crawlPage(page *crawlerdb.Page) ([]string, error) {
	var urls []string
	resp, err := getRequest(page.URL)
	if err != nil {
		return urls, err
	}
	paths := findRawURLs(resp)
	urls, err = filterURLs(paths, page.URL)
	if err != nil {
		return urls, err
	}
	err = c.db.UpdatePageEdges(page.ID, urls)
	if err != nil {
		return urls, err
	}
	return urls, nil
}

// nextPagesFromEdges grabs next pages using already existing edges in the graph
// and returns a slice of strings representing urls for the next pages.
func (c *GraphCrawler) nextPagesFromEdges(page *crawlerdb.Page) ([]string, error) {
	var urls []string
	edges, err := c.db.GetEdgesForPage(page)
	if err != nil {
		return urls, err
	}
	for _, e := range edges {
		nextTaskPage, err := c.db.GetPage(e.TargetID)
		if err != nil {
			return urls, err
		}
		urls = append(urls, nextTaskPage.URL)
	}
	return urls, nil
}

// addNewTasks adds new tasks to the database, if necessary.
func (c *GraphCrawler) addNewTasks(t *crawlerdb.Task, levels int, urls []string) error {
	if t.CurrentLevel < levels {
		fmt.Printf("Adding %d new tasks.\n", len(urls))
		for _, u := range urls {
			err := c.db.CreateTask(t.CrawlRequestID, u, t.CurrentLevel+1)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}
	}
	return nil
}

// handleError prints out an informative error message and sets the task status
// to FAILED in the event of an error.
func (c *GraphCrawler) handleError(t *crawlerdb.Task, err error) {
	fmt.Printf("Error while crawling task %d (url %s) at level %d for CrawlRequest %v: %s\n", t.ID, t.PageURL, t.CurrentLevel, t.CrawlRequestID, err)
	if t != nil {
		err = c.db.UpdateTaskStatus(t.ID, "FAILED")
		if err != nil {
			fmt.Printf("Error while updating task %d (url %s) at level %d for CrawlRequest %v: %s\n", t.ID, t.PageURL, t.CurrentLevel, t.CrawlRequestID, err)
		}
	}
	// debug.PrintStack()
}
