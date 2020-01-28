package api

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/emilyzhang/crawlr/crawlerdb"
)

// Server represents an API server containing a database client and a logger.
type Server struct {
	Logger *log.Logger
	db     *crawlerdb.Postgres
}

// New creates a new API Server.
func New(dbDSN string) (*Server, error) {
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

	return &Server{
		Logger: log.New(os.Stdout, "", 0),
		db:     db,
	}, nil
}

// Start starts the server.
func (s *Server) Start() {
	http.HandleFunc("/", s.router)
	s.Logger.Print("Starting API server. Hello world!")
	http.ListenAndServe(":8000", nil)
	// TODO: implement graceful server shut down
}
