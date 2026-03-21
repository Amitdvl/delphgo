package main

import (
	"strings"
	"time"
)

// FilterOptions controls which topics are included in output.
type FilterOptions struct {
	Category string
	Keyword  string
	Tags     []string
	Since    time.Duration
}

// FilterTopics returns topics matching all active filter criteria.
func FilterTopics(topics []Topic, opts FilterOptions) []Topic {
	if opts.Category == "" && opts.Keyword == "" && len(opts.Tags) == 0 && opts.Since == 0 {
		return topics
	}

	cutoff := time.Time{}
	if opts.Since > 0 {
		cutoff = time.Now().Add(-opts.Since)
	}

	result := make([]Topic, 0, len(topics))
	for _, t := range topics {
		if !matchesTopic(t, opts, cutoff) {
			continue
		}
		result = append(result, t)
	}
	return result
}

func matchesTopic(t Topic, opts FilterOptions, cutoff time.Time) bool {
	if opts.Category != "" {
		if !strings.EqualFold(t.Category, opts.Category) {
			return false
		}
	}

	if opts.Keyword != "" {
		kw := strings.ToLower(opts.Keyword)
		inTitle := strings.Contains(strings.ToLower(t.Title), kw)
		inDesc := strings.Contains(strings.ToLower(t.Description), kw)
		inAuthor := strings.Contains(strings.ToLower(t.Author), kw)
		if !inTitle && !inDesc && !inAuthor {
			return false
		}
	}

	if len(opts.Tags) > 0 {
		if !hasAnyTag(t.Tags, opts.Tags) {
			return false
		}
	}

	if !cutoff.IsZero() && !t.PublishedAt.IsZero() {
		if t.PublishedAt.Before(cutoff) {
			return false
		}
	}

	return true
}

func hasAnyTag(topicTags, filterTags []string) bool {
	tagSet := make(map[string]bool, len(topicTags))
	for _, tag := range topicTags {
		tagSet[strings.ToLower(tag)] = true
	}
	for _, ft := range filterTags {
		if tagSet[strings.ToLower(ft)] {
			return true
		}
	}
	return false
}
