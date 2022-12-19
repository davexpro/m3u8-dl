package util

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

var (
	httpClient = &http.Client{
		Timeout: time.Minute,
	}
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:108.0) Gecko/20100101 Firefox/108.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13.1; rv:108.0) Gecko/20100101 Firefox/108.0",
		"Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	}
	Origin    string
	Referer   string
	Cookies   string
	UserAgent string
)

func Get(url string) (io.ReadCloser, error) {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if len(UserAgent) > 0 {
		req.Header.Set("user-agent", UserAgent)
	} else {
		req.Header.Set("user-agent", userAgents[rand.Intn(len(userAgents))])
	}

	if len(Origin) > 0 {
		req.Header.Set("origin", Origin)
	}
	if len(Referer) > 0 {
		req.Header.Set("referer", Referer)
	}
	if len(Cookies) > 0 {
		req.Header.Set("cookie", Cookies)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http error: status code %d", resp.StatusCode)
	}
	return resp.Body, nil
}
