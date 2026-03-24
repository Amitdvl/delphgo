package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	userAgent    = "delphgo/1.0 (read-only summarizer)"
	maxFeedBytes = 2 << 20 // 2 MB
	maxPageBytes = 8 << 20 // 8 MB
	cdHost       = "https://www.chiefdelphi.com"
)

const defaultFeedURL = "https://www.chiefdelphi.com/latest.rss"

// rssFeed mirrors the Discourse RSS XML structure.
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
	Creator     string `xml:"creator"`
	Category    string `xml:"category"`
}

// FetchFeed downloads and parses the Chief Delphi RSS feed.
func FetchFeed(feedURL string) ([]Topic, error) {
	if feedURL == "" {
		feedURL = defaultFeedURL
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("building feed request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFeedBytes))
	if err != nil {
		return nil, fmt.Errorf("reading feed body: %w", err)
	}

	return parseFeed(body)
}

func parseFeed(data []byte) ([]Topic, error) {
	var feed rssFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parsing RSS XML: %w", err)
	}

	topics := make([]Topic, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		// Only accept links from the expected host to prevent poisoned-feed injection.
		link := item.Link
		if !strings.HasPrefix(link, cdHost+"/") {
			link = ""
		}

		t := Topic{
			Title:       item.Title,
			Link:        link,
			Author:      item.Creator,
			Category:    item.Category,
			Description: item.Description,
		}

		if pub, err := parseRSSDate(item.PubDate); err == nil {
			t.PublishedAt = pub
		}

		topics = append(topics, t)
	}

	return topics, nil
}

// parseRSSDate handles common RSS date formats.
func parseRSSDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %q", s)
}
