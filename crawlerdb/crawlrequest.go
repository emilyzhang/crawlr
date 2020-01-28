package crawlerdb

import (
	"fmt"
	"net/url"

	_ "github.com/jackc/pgx/v4/stdlib"
)

// CreateCrawlRequest creates a new crawl request.
func (p *Postgres) CreateCrawlRequest(urlString string, levels int) (int, error) {
	var id int
	// clean url and make sure there's a scheme attached
	pageURL, err := url.Parse(urlString)
	if err != nil {
		return id, fmt.Errorf("Unable to parse url %s: %v", urlString, err)
	}
	pageURL.Fragment = ""
	if pageURL.Scheme == "" {
		pageURL.Scheme = "http"
	}

	// create new crawl request
	result := p.db.QueryRow(
		`INSERT INTO crawl_requests
		(id, url, levels)
		VALUES (DEFAULT, $1, $2)
		RETURNING id`, pageURL.String(), levels)
	err = result.Scan(&id)
	if err != nil {
		return id, fmt.Errorf("Unable to create crawl request with url %s and level %d: %v", urlString, levels, err)
	}
	// create first task for crawl request
	err = p.CreateTask(id, pageURL.String(), 0)
	if err != nil {
		return id, fmt.Errorf("Unable to create task for crawl request %d: %v", id, err)
	}
	return id, err
}

// GetCrawlRequest gets the crawl request associated with the given id.
func (p *Postgres) GetCrawlRequest(id int) (*CrawlRequest, error) {
	var cr CrawlRequest
	result := p.db.QueryRow(
		`SELECT id, url, levels
			FROM crawl_requests
			WHERE id = $1`, id)
	err := result.Scan(&cr.ID, &cr.URL, &cr.Levels)
	if err != nil {
		return nil, fmt.Errorf("Unable to get crawl request with id %d: %v", id, err)
	}
	return &cr, nil
}

// CrawlRequestStatus returns information related to the status of a crawl
// request.
func (p *Postgres) CrawlRequestStatus(crawlRequestID int) (*CrawlRequestStatus, error) {
	var crs CrawlRequestStatus
	rows, err := p.db.Query(
		`SELECT status, COUNT(*)
		FROM tasks
		WHERE crawl_request_id = $1
		AND current_level < (SELECT levels FROM crawl_requests WHERE id = $1)
		GROUP BY status`, crawlRequestID)
	if err != nil {
		return nil, fmt.Errorf("Unable to get tasks for crawl request with id %d: %v", crawlRequestID, err)
	}
	defer rows.Close()

	// get status counts
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("Unable to scan status for crawl request with id %d: %v", crawlRequestID, err)
		}
		switch status {
		case "COMPLETED":
			crs.Completed = count
		case "IN_PROGRESS":
			crs.InProgress = count
		case "FAILED":
			crs.Failed = count
		}
	}
	return &crs, nil
}

// GetCrawlRequestTasks returns all tasks crawled during a crawl request. If all URLs
// have completed crawling, it is likely the crawl request is done.
func (p *Postgres) GetCrawlRequestTasks(crawlRequestID int) ([]*Task, error) {
	var tasks []*Task
	rows, err := p.db.Query(
		`SELECT id, crawl_request_id, page_url, current_level, status
		FROM tasks
		WHERE crawl_request_id = $1`, crawlRequestID)
	if err != nil {
		return tasks, fmt.Errorf("Unable to get tasks for crawl request with id %d: %v", crawlRequestID, err)
	}
	defer rows.Close()

	for rows.Next() {
		t := Task{}
		if err := rows.Scan(&t.ID, &t.CrawlRequestID, &t.PageURL, &t.CurrentLevel, &t.Status); err != nil {
			return tasks, fmt.Errorf("Unable to scan tasks for crawl request with id %d: %v", crawlRequestID, err)
		}
		tasks = append(tasks, &t)
	}
	return tasks, nil
}
