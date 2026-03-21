package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ScrapeTopic fetches a topic page and extracts posts and metadata.
// Discourse serves a crawler-friendly HTML view with posts in
// div.crawler-post containers using Schema.org microdata.
func ScrapeTopic(topicURL string) ([]Post, int, int, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, topicURL, nil)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("building topic request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("fetching topic: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, 0, 0, fmt.Errorf("topic page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes))
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("reading topic body: %w", err)
	}

	return parseTopic(string(body))
}

func parseTopic(rawHTML string) ([]Post, int, int, int, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("parsing HTML: %w", err)
	}

	var posts []Post

	// Find all div.crawler-post elements
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "crawler-post") {
			post := extractCrawlerPost(n)
			if post.Content != "" {
				posts = append(posts, post)
			}
			return // don't recurse into crawler-post children
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Discourse crawler view doesn't include view/like counts;
	// reply count is inferred from post count in main.go
	return posts, 0, 0, 0, nil
}

// extractCrawlerPost extracts author and content from a Discourse crawler-post div.
// Structure: div.crawler-post > div.crawler-post-meta > span.creator > a > span[itemprop=name]
//
//	div.crawler-post > div.post[itemprop=text]
func extractCrawlerPost(n *html.Node) Post {
	var post Post

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}

		// Extract author from crawler-post-meta
		if hasClass(c, "crawler-post-meta") {
			post.Author = findItempropName(c)
			post.CreatedAt = findPostTime(c)
		}

		// Extract content from div.post[itemprop=text]
		if c.Data == "div" && hasClass(c, "post") && getAttr(c, "itemprop") == "text" {
			post.Content = strings.TrimSpace(extractText(c))
		}
	}

	return post
}

// findItempropName finds the text of span[itemprop="name"] within a subtree.
func findItempropName(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "span" && getAttr(n, "itemprop") == "name" {
		return strings.TrimSpace(extractText(n))
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if name := findItempropName(c); name != "" {
			return name
		}
	}
	return ""
}

// findPostTime extracts the time from a meta or time element in the crawler-post-meta.
func findPostTime(n *html.Node) time.Time {
	if n.Type == html.ElementNode {
		if n.Data == "time" || n.Data == "meta" {
			dt := getAttr(n, "datetime")
			if dt == "" {
				dt = getAttr(n, "content")
			}
			if dt != "" {
				if t, err := time.Parse(time.RFC3339, dt); err == nil {
					return t
				}
				if t, err := time.Parse("2006-01-02T15:04:05Z", dt); err == nil {
					return t
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := findPostTime(c); !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

// extractText returns all text content under a node.
func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	classes := getAttr(n, "class")
	for _, c := range strings.Fields(classes) {
		if c == class {
			return true
		}
	}
	return false
}
