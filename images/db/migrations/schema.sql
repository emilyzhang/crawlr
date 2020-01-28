CREATE TABLE page_nodes (
    id             SERIAL PRIMARY KEY,
    url            TEXT NOT NULL UNIQUE,
    crawled_status BOOLEAN NOT NULL
);

CREATE TABLE crawl_requests (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    levels INTEGER NOT NULL
);

CREATE TABLE edges (
    id           SERIAL PRIMARY KEY,
    source_id    INTEGER NOT NULL REFERENCES page_nodes(id),
    target_id    INTEGER NOT NULL REFERENCES page_nodes(id),
    CHECK (source_id != target_id)
);

CREATE TABLE tasks (
    id               SERIAL PRIMARY KEY,
    crawl_request_id INTEGER NOT NULL REFERENCES crawl_requests(id),
    page_url         TEXT NOT NULL,
    current_level    INTEGER NOT NULL,
    status           TEXT NOT NULL,
    UNIQUE(crawl_request_id, page_url)
);
