package crawlerdb

import (
	"fmt"
)

// UpsertPage creates a new page node if a page node with that url doesn't
// already exist, otherwise it returns the existing page node.
func (p *Postgres) UpsertPage(url string) (int, error) {
	var id int
	result := p.db.QueryRow(
		`INSERT INTO page_nodes
		(id, url, crawled_status)
		VALUES (DEFAULT, $1, $2)
		ON CONFLICT (url) DO UPDATE SET url=$1
		RETURNING id`, url, false)
	err := result.Scan(&id)
	if err != nil {
		return id, fmt.Errorf("Unable to upsert page with url %s: %v", url, err)
	}
	return id, err
}

// GetPage returns the page node associated with the given id.
func (p *Postgres) GetPage(id int) (*Page, error) {
	var page Page
	result := p.db.QueryRow(
		`SELECT id, url, crawled_status
		FROM page_nodes
		WHERE id = $1`, id)
	err := result.Scan(&page.ID, &page.URL, &page.CrawledStatus)
	if err != nil {
		return nil, fmt.Errorf("Unable to get page %d: %v", id, err)
	}
	return &page, nil
}

// GetEdgesForPage returns all edges associated with a page where the given page
// is the source node.
func (p *Postgres) GetEdgesForPage(page *Page) ([]Edge, error) {
	var edges []Edge
	rows, err := p.db.Query(
		`SELECT id, source_id, target_id
		FROM edges
		WHERE source_id = $1`, page.ID)
	if err != nil {
		return edges, fmt.Errorf("Unable to retrieve edges for page %d: %v", page.ID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, sourceID, targetID int
		if err := rows.Scan(&id, &sourceID, &targetID); err != nil {
			return edges, err
		}
		edges = append(edges, Edge{ID: id, SourceID: sourceID, TargetID: targetID})
	}
	return edges, nil
}

// UpdatePageEdges adds new edges for a page node, creating new pages in the
// process if necessary. It also updates the CrawledStatus of the given page
// node to true.
func (p *Postgres) UpdatePageEdges(pageID int, urls []string) error {
	for _, url := range urls {
		targetID, err := p.UpsertPage(url)
		if err != nil {
			return fmt.Errorf("Could not upsert page with url %s during edge update: %v", url, err)
		}
		if targetID != pageID {
			// add edge to graph
			_, err = p.db.Exec(
				`INSERT INTO edges
				(id, source_id, target_id)
				VALUES (DEFAULT, $1, $2)
				ON CONFLICT DO NOTHING`, pageID, targetID)
			if err != nil {
				fmt.Printf("Could not insert edge between source page %d and target page %d during edge update: %v", pageID, targetID, err)
				continue
			}
		}
	}

	// update crawled status of page to true
	_, err := p.db.Exec(
		`UPDATE page_nodes
		SET crawled_status=$1
		WHERE id=$2`, true, pageID)
	if err != nil {
		return fmt.Errorf("Unable to update page %d status to true: %v", pageID, err)
	}
	return nil
}
