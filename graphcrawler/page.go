package graphcrawler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/html"
)

// getRequest returns an *http.Response for a given url.
func getRequest(url string) (*http.Response, error) {
	cl := http.Client{Timeout: 2 * time.Minute}
	resp, err := cl.Get(url)
	if err != nil {
		return resp, err
	} else if resp.StatusCode != 200 {
		// for now, this ignores redirects & other possible non-error http
		// status codes
		fmt.Printf("Got status code: %d %v when crawling %s\n", resp.StatusCode, http.StatusText(resp.StatusCode), url)
		return resp, errors.New("Received a non-200 status code")
	}
	return resp, nil
}

// findRawURLs uses an html parser to find links from html tags.
func findRawURLs(resp *http.Response) []string {
	defer resp.Body.Close()
	var paths []string
	done := false
	tokenizer := html.NewTokenizer(resp.Body)
	for !done {
		t := tokenizer.Next()
		switch t {
		case html.ErrorToken:
			done = true
		case html.StartTagToken:
			token := tokenizer.Token()
			// at this time it ignores all other types of tags
			if token.Data == "a" {
				for _, a := range token.Attr {
					if a.Key == "href" {
						paths = append(paths, a.Val)
						break
					}
				}
			}
		}
	}
	return paths
}

// filterURLs filters through a slice of strings representing urls, removing any
// urls that can't be parsed or don't have the proper protocols.
func filterURLs(links []string, refURL string) ([]string, error) {
	var urls []string
	// parse ref url
	ref, err := url.Parse(refURL)
	if err != nil {
		return urls, err
	}

	for _, path := range links {
		// strip fragments from the url
		stripFragment, err := url.Parse(path)
		if err != nil {
			fmt.Printf("Unable to parse path into URL: %s", path)
			continue
		}
		stripFragment.Fragment = ""
		path = stripFragment.String()
		// parse relative urls based on ref
		u, err := ref.Parse(path)
		if err != nil {
			fmt.Printf("Unable to parse path into URL: %s", u)
			continue
		}
		// ignore urls that don't have http or https protocol set
		// (ex: mailto or ftp protocols)
		if u.Scheme != "http" && u.Scheme != "https" {
			continue
		}
		urls = append(urls, u.String())
	}
	return urls, nil
}
