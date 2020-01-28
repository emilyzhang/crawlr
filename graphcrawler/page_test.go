package graphcrawler

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLParsing(t *testing.T) {
	t.Run("successfully retrieves http response from url", func(tt *testing.T) {
		_, err := getRequest("http://google.com")
		assert.NoError(tt, err)
	})

	t.Run("successfully finds all expected raw urls from page", func(tt *testing.T) {
		resp := &http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(`<body><a href="index.html">origin</a><a href="<http://support.com>">support</a><a href="<http://google.com>">search<a><a href="<https://support.com/example>">support<a></body>`))}
		urls := findRawURLs(resp)
		assert.Len(tt, urls, 4)
	})

	t.Run("successfully parses raw urls to expected format, stripping fragments", func(tt *testing.T) {
		resp := &http.Response{Body: ioutil.NopCloser(bytes.NewBufferString(`<body><a href="index.html">origin</a><a href="http://support.com/#hello">support</a><a href="http://google.com">search<a><a href="https://support.com/example">support<a></body>`))}
		urls := findRawURLs(resp)
		urls, err := filterURLs(urls, "http://example.com/about")
		assert.NoError(tt, err)
		assert.Len(tt, urls, 4)
		// parses relative urls correctly
		assert.Equal(tt, "http://example.com/index.html", urls[0])
		// strips fragment correctly
		assert.Equal(tt, "http://support.com/", urls[1])
	})

}
