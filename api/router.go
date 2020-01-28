package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/emilyzhang/crawlr/crawlerdb"
)

// router routes requests to the correct handler.
func (s *Server) router(w http.ResponseWriter, req *http.Request) {
	s.Logger.Printf("New request: %s", req.URL.Path)
	pathPattern := regexp.MustCompile(`/(status|results)/\d+`)
	if req.URL.Path == "/crawl" && req.Method == http.MethodPost {
		s.createHandler(w, req)
		return
	} else if pathPattern.MatchString(req.URL.Path) && req.Method == http.MethodGet {
		u := strings.Split("/"+path.Clean(req.URL.Path), "/")
		id, err := strconv.Atoi(u[3])
		if err != nil {
			http.Error(w, "invalid id: "+u[3], http.StatusBadRequest)
		}
		switch u[2] {
		case "status":
			s.statusHandler(w, req, id)
		case "results":
			s.resultsHandler(w, req, id)
		}
	} else {
		http.Error(w, "Not a valid endpoint: "+req.URL.Path, http.StatusNotFound)
	}
}

// createHandler specifies a handler for the / endpoint.
func (s *Server) createHandler(w http.ResponseWriter, req *http.Request) {
	c := &struct {
		URL    string
		Levels int
	}{}

	// Read request body.
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		s.Logger.Printf("Error from request %s: %s", req.URL.Path, err.Error())
	}

	err = json.Unmarshal(body, c)
	if err != nil {
		s.Logger.Printf("Error from request %s: %s", req.URL.Path, err.Error())
	}

	id, err := s.db.CreateCrawlRequest(c.URL, c.Levels)
	if err != nil {
		w.Write([]byte("Error while creating request: " + err.Error()))
		s.Logger.Printf("Error from request %s: %s", req.URL.Path, err.Error())
		return
	}
	resp := fmt.Sprintf("Created new crawl request starting from %s with %d levels.\nRequest ID: %d", c.URL, c.Levels, id)
	w.Write([]byte(resp))
}

// statusHandler specifies a handler for the /status/<id> endpoint.
func (s *Server) statusHandler(w http.ResponseWriter, req *http.Request, id int) {
	cr, err := s.db.GetCrawlRequest(id)
	if err != nil {
		if err == crawlerdb.ErrDoesNotExist {
			http.Error(w, "There is no crawl request with this id.", http.StatusInternalServerError)
		} else {
			http.Error(w, err.Error()+".", http.StatusInternalServerError)
		}
		return
	}
	crs, err := s.db.CrawlRequestStatus(id)
	if err != nil {
		http.Error(w, err.Error()+".", http.StatusInternalServerError)
		return
	}
	var status string
	total := crs.InProgress + crs.Completed + crs.Failed
	if total == 0 {
		status = fmt.Sprintf("CrawlRequest %d not started yet.", id)
	} else {
		status = fmt.Sprintf("Starting from %s: Completed %d URLs. Crawling in progress on %d URLs. Failed to crawl %d URLs. Total attempted crawls: %d", cr.URL, crs.Completed, crs.InProgress, crs.Failed, total)
	}
	w.Write([]byte(status))
}

// resultsHandler specifies a handler for the /results/<id> endpoint.
func (s *Server) resultsHandler(w http.ResponseWriter, req *http.Request, id int) {
	cr, err := s.db.GetCrawlRequest(id)
	if err != nil {
		if err == crawlerdb.ErrDoesNotExist {
			http.Error(w, "There is no crawl request with this id.", http.StatusInternalServerError)
		} else {
			http.Error(w, err.Error()+".", http.StatusInternalServerError)
		}
		return
	}
	tasks, err := s.db.GetCrawlRequestTasks(id)
	if err != nil {
		http.Error(w, "Could not retrieve results: "+err.Error(), http.StatusInternalServerError)
		return
	}
	hosts, err := s.hostCounter(tasks, id, cr.URL)
	if err != nil {
		http.Error(w, "Could not retrieve results: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h, err := json.Marshal(hosts)
	if err != nil {
		http.Error(w, "Could unmarshal results into json: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Hosts found: " + string(h)))
}

// hostCounter is a helperfunction for resultsHandler that takes in a list of
// tasks and returns a count of all hosts traversed during those tasks. Returns
// a nil map if CrawlRequest is not yet done.
func (s *Server) hostCounter(tasks []*crawlerdb.Task, crawlRequestID int, originalURL string) (map[string]int, error) {
	hosts := make(map[string]int, 0)
	o, err := url.Parse(originalURL)
	if err != nil {
		s.Logger.Printf("CrawlRequest %d: Error parsing original url %s: %v", crawlRequestID, originalURL, err.Error())
	}
	originalHost := o.Hostname()
	for _, t := range tasks {
		if t.Status == "IN_PROGRESS" || t.Status == "NOT_STARTED" {
			return hosts, errors.New("CrawlRequest not yet completed")
		}
		u, err := url.Parse(t.PageURL)
		if err != nil {
			s.Logger.Printf("CrawlRequest %d: Error parsing url %s for task %d: %v", t.CrawlRequestID, t.PageURL, t.ID, err.Error())
		}
		// don't add to results count if the host is the same as the original given host
		if host := u.Hostname(); host != originalHost {
			if _, ok := hosts[host]; ok {
				hosts[host]++
			} else {
				hosts[host] = 1
			}
		}
	}
	return hosts, nil
}
