package crawlerdb

// Page represents a page node in the crawler graph.
type Page struct {
	ID            int
	URL           string
	CrawledStatus bool
}

// CrawlRequest represents a single crawl request.
type CrawlRequest struct {
	ID     int
	URL    string
	Levels int
}

// Edge represents an edge between a source Page and a target Page.
type Edge struct {
	ID       int
	SourceID int
	TargetID int
}

// Task represents a page to be crawled.
type Task struct {
	ID             int
	CrawlRequestID int
	PageURL        string
	CurrentLevel   int
	Status         string
}

// CrawlRequestStatus represents the status of a CrawlRequest.
type CrawlRequestStatus struct {
	Completed  int
	Failed     int
	InProgress int
}
