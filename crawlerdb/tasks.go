package crawlerdb

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNoTasksAvailable = errors.New("no tasks available right now")
	ErrDoesNotExist     = errors.New("sql: no rows in result set")
)

// CreateTask creates a new task.
func (p *Postgres) CreateTask(crawlRequestID int, url string, currLevel int, seen bool) error {
	_, err := p.db.Exec(
		`INSERT INTO tasks
		(id, crawl_request_id, page_url, current_level, status, seen_url)
		VALUES (DEFAULT, $1, $2, $3, $4, $5)`, crawlRequestID, url, currLevel, "NOT_STARTED", seen)
	if err != nil {
		return fmt.Errorf("Unable to create task: %v", err)
	}
	return nil
}

// UpdateTaskStatus updates the status of a task.
func (p *Postgres) UpdateTaskStatus(id int, status string) error {
	_, err := p.db.Exec(
		`UPDATE tasks
		SET status = $2
		WHERE id = $1`, id, status)
	if err != nil {
		return fmt.Errorf("Unable to update task: %v", err)
	}
	return err
}

// FindIncompleteTask finds and returns the first task (sorted by
// crawl_request_id ascending) that has not been started yet. It also updates
// the task status to "IN_PROGRESS".
func (p *Postgres) FindIncompleteTask() (*Task, error) {
	var t Task
	result := p.db.QueryRow(
		`UPDATE tasks
		SET status = $1
		WHERE id = (SELECT id FROM tasks
			WHERE status=$2
			ORDER BY crawl_request_id ASC
			LIMIT 1)
		RETURNING id, crawl_request_id, page_url, current_level, status, seen_url`, "IN_PROGRESS", "NOT_STARTED")
	err := result.Scan(&t.ID, &t.CrawlRequestID, &t.PageURL, &t.CurrentLevel, &t.Status, &t.SeenURL)
	if err == sql.ErrNoRows {
		return nil, ErrNoTasksAvailable
	}
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve next task: %v", err)
	}
	return &t, nil
}
