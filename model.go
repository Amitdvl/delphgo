package main

import "time"

// Topic represents a Chief Delphi forum topic.
type Topic struct {
	Title       string
	Link        string
	Author      string
	Category    string
	Tags        []string
	PublishedAt time.Time
	Description string // HTML content from RSS (first post)

	// Populated by deep scrape
	ReplyCount int
	ViewCount  int
	LikeCount  int
	Posts      []Post
}

// Post represents a single reply within a topic.
type Post struct {
	Author    string
	Content   string // plain text
	CreatedAt time.Time
	LikeCount int
}
