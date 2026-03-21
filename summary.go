package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// GenerateSummary produces a Markdown summary of topics.
func GenerateSummary(topics []Topic) string {
	if len(topics) == 0 {
		return "# Chief Delphi Summary\n\nNo topics found matching the given criteria.\n"
	}

	var sb strings.Builder

	sb.WriteString("# Chief Delphi Summary\n\n")
	sb.WriteString(fmt.Sprintf("*Generated %s — %d topics analyzed*\n\n",
		time.Now().Format("2006-01-02 15:04 MST"), len(topics)))

	sb.WriteString("---\n\n")

	// Section 1: Main Themes (group by category)
	writeThemes(&sb, topics)

	// Section 2: Important Threads (highest engagement)
	writeImportantThreads(&sb, topics)

	// Section 3: Actionable Takeaways
	writeActionable(&sb, topics)

	// Section 4: Useful Links
	writeLinks(&sb, topics)

	// Section 5: Repeated Opinions / Sentiment
	writeSentiment(&sb, topics)

	return sb.String()
}

func writeThemes(sb *strings.Builder, topics []Topic) {
	sb.WriteString("## Main Themes\n\n")

	byCategory := groupByCategory(topics)

	// Sort categories by count (descending)
	type catCount struct {
		name  string
		count int
	}
	cats := make([]catCount, 0, len(byCategory))
	for cat, ts := range byCategory {
		cats = append(cats, catCount{cat, len(ts)})
	}
	sort.Slice(cats, func(i, j int) bool {
		return cats[i].count > cats[j].count
	})

	for _, cc := range cats {
		label := cc.name
		if label == "" {
			label = "Uncategorized"
		}
		sb.WriteString(fmt.Sprintf("### %s (%d topics)\n\n", label, cc.count))
		for _, t := range byCategory[cc.name] {
			timeStr := ""
			if !t.PublishedAt.IsZero() {
				timeStr = fmt.Sprintf(" — %s", t.PublishedAt.Format("Jan 2 15:04"))
			}
			engagement := ""
			if t.ReplyCount > 0 || t.ViewCount > 0 {
				parts := []string{}
				if t.ReplyCount > 0 {
					parts = append(parts, fmt.Sprintf("%d replies", t.ReplyCount))
				}
				if t.ViewCount > 0 {
					parts = append(parts, fmt.Sprintf("%d views", t.ViewCount))
				}
				if t.LikeCount > 0 {
					parts = append(parts, fmt.Sprintf("%d likes", t.LikeCount))
				}
				engagement = fmt.Sprintf(" [%s]", strings.Join(parts, ", "))
			}
			sb.WriteString(fmt.Sprintf("- **%s** by %s%s%s\n",
				t.Title, authorOrUnknown(t.Author), timeStr, engagement))
		}
		sb.WriteString("\n")
	}
}

func writeImportantThreads(sb *strings.Builder, topics []Topic) {
	sb.WriteString("## Important Threads\n\n")

	// Sort by engagement (reply count, then view count)
	sorted := make([]Topic, len(topics))
	copy(sorted, topics)
	sort.Slice(sorted, func(i, j int) bool {
		scoreI := sorted[i].ReplyCount*3 + sorted[i].ViewCount + sorted[i].LikeCount*2
		scoreJ := sorted[j].ReplyCount*3 + sorted[j].ViewCount + sorted[j].LikeCount*2
		return scoreI > scoreJ
	})

	// Show top threads (up to 10, or all if few)
	limit := 10
	if limit > len(sorted) {
		limit = len(sorted)
	}

	hasEngagement := false
	for i := 0; i < limit; i++ {
		t := sorted[i]
		if t.ReplyCount > 0 || t.ViewCount > 0 {
			hasEngagement = true
			sb.WriteString(fmt.Sprintf("1. **[%s](%s)**\n", t.Title, t.Link))
			meta := []string{fmt.Sprintf("by %s", authorOrUnknown(t.Author))}
			if t.ReplyCount > 0 {
				meta = append(meta, fmt.Sprintf("%d replies", t.ReplyCount))
			}
			if t.ViewCount > 0 {
				meta = append(meta, fmt.Sprintf("%d views", t.ViewCount))
			}
			if t.LikeCount > 0 {
				meta = append(meta, fmt.Sprintf("%d likes", t.LikeCount))
			}
			sb.WriteString(fmt.Sprintf("   - %s\n", strings.Join(meta, " | ")))

			// Include snippet from first post if available
			snippet := firstSentences(stripHTML(t.Description), 2)
			if snippet != "" {
				sb.WriteString(fmt.Sprintf("   - *%s*\n", snippet))
			}

			// Show notable replies if deep-scraped
			if len(t.Posts) > 1 {
				shown := 0
				for _, p := range t.Posts[1:] {
					if shown >= 2 {
						break
					}
					replySnippet := firstSentences(p.Content, 1)
					if replySnippet != "" {
						author := authorOrUnknown(p.Author)
						sb.WriteString(fmt.Sprintf("   - > **%s**: %s\n", author, replySnippet))
						shown++
					}
				}
			}
		}
	}

	if !hasEngagement {
		// Fall back: list first few by recency with snippets
		sb.WriteString("*Engagement data not available (use `--deep` to fetch). Showing most recent:*\n\n")
		for i := 0; i < limit; i++ {
			t := sorted[i]
			sb.WriteString(fmt.Sprintf("1. **[%s](%s)** by %s\n", t.Title, t.Link, authorOrUnknown(t.Author)))
			snippet := firstSentences(stripHTML(t.Description), 2)
			if snippet != "" {
				sb.WriteString(fmt.Sprintf("   - *%s*\n", snippet))
			}
		}
	}
	sb.WriteString("\n")
}

func writeActionable(sb *strings.Builder, topics []Topic) {
	sb.WriteString("## Actionable Takeaways\n\n")

	// Look for topics with actionable keywords in title or description
	actionWords := []string{
		"help", "needed", "looking for", "wanted", "how to", "question",
		"issue", "problem", "fix", "update", "announcement", "release",
		"deadline", "event", "registration", "schedule", "rule", "change",
	}

	found := false
	for _, t := range topics {
		lower := strings.ToLower(t.Title + " " + stripHTML(t.Description))
		for _, aw := range actionWords {
			if strings.Contains(lower, aw) {
				sb.WriteString(fmt.Sprintf("- [%s](%s) — %s\n",
					t.Title, t.Link, categorizeAction(lower)))
				found = true
				break
			}
		}
	}

	if !found {
		sb.WriteString("*No clearly actionable items identified in current topics.*\n")
	}
	sb.WriteString("\n")
}

func writeLinks(sb *strings.Builder, topics []Topic) {
	sb.WriteString("## Useful Links\n\n")

	for _, t := range topics {
		sb.WriteString(fmt.Sprintf("- [%s](%s)", t.Title, t.Link))
		if t.Category != "" {
			sb.WriteString(fmt.Sprintf(" `%s`", t.Category))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
}

func writeSentiment(sb *strings.Builder, topics []Topic) {
	sb.WriteString("## Repeated Opinions & Sentiment\n\n")

	// Simple word frequency analysis on titles to find recurring themes
	wordFreq := make(map[string]int)
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"is": true, "in": true, "to": true, "for": true, "of": true,
		"on": true, "at": true, "by": true, "it": true, "we": true,
		"our": true, "with": true, "this": true, "that": true, "from": true,
		"be": true, "are": true, "was": true, "were": true, "been": true,
		"has": true, "have": true, "had": true, "do": true, "does": true,
		"not": true, "but": true, "if": true, "so": true, "as": true,
		"what": true, "how": true, "when": true, "where": true, "who": true,
		"which": true, "will": true, "can": true, "i": true, "you": true,
		"my": true, "your": true, "he": true, "she": true, "they": true,
		"all": true, "about": true, "up": true, "out": true, "just": true,
		"than": true, "them": true, "then": true, "no": true, "yes": true,
		"its": true, "more": true, "any": true, "some": true, "new": true,
		"-": true, "–": true, "—": true, "|": true, "": true,
	}

	for _, t := range topics {
		words := strings.Fields(strings.ToLower(t.Title))
		for _, w := range words {
			w = strings.Trim(w, ".,!?:;()[]\"'")
			if len(w) < 3 || stopWords[w] {
				continue
			}
			wordFreq[w]++
		}
	}

	type wf struct {
		word  string
		count int
	}
	var sorted []wf
	for w, c := range wordFreq {
		if c >= 2 { // only words appearing 2+ times
			sorted = append(sorted, wf{w, c})
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	if len(sorted) == 0 {
		sb.WriteString("*Not enough data to identify recurring themes.*\n")
	} else {
		sb.WriteString("Recurring terms across topic titles:\n\n")
		limit := 15
		if limit > len(sorted) {
			limit = len(sorted)
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("- **%s** (%dx)\n", sorted[i].word, sorted[i].count))
		}
	}
	sb.WriteString("\n")
}

// --- helpers ---

func groupByCategory(topics []Topic) map[string][]Topic {
	m := make(map[string][]Topic)
	for _, t := range topics {
		m[t.Category] = append(m[t.Category], t)
	}
	return m
}

func authorOrUnknown(a string) string {
	if a == "" {
		return "unknown"
	}
	return a
}

func stripHTML(s string) string {
	var sb strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			sb.WriteRune(' ')
			continue
		}
		if !inTag {
			sb.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

func firstSentences(text string, n int) string {
	if text == "" {
		return ""
	}
	// Truncate to reasonable length first
	if len(text) > 500 {
		text = text[:500]
	}
	sentences := 0
	for i, r := range text {
		if r == '.' || r == '!' || r == '?' {
			sentences++
			if sentences >= n {
				return strings.TrimSpace(text[:i+1])
			}
		}
	}
	// If not enough sentence-ending punctuation, just return truncated text
	if len(text) > 200 {
		return strings.TrimSpace(text[:200]) + "..."
	}
	return strings.TrimSpace(text)
}

func categorizeAction(lower string) string {
	switch {
	case strings.Contains(lower, "help") || strings.Contains(lower, "needed") || strings.Contains(lower, "looking for"):
		return "help/request"
	case strings.Contains(lower, "event") || strings.Contains(lower, "registration") || strings.Contains(lower, "schedule"):
		return "event/deadline"
	case strings.Contains(lower, "announcement") || strings.Contains(lower, "release") || strings.Contains(lower, "update"):
		return "announcement/update"
	case strings.Contains(lower, "rule") || strings.Contains(lower, "change"):
		return "rule change"
	case strings.Contains(lower, "issue") || strings.Contains(lower, "problem") || strings.Contains(lower, "fix"):
		return "issue/troubleshooting"
	case strings.Contains(lower, "question") || strings.Contains(lower, "how to"):
		return "question"
	default:
		return "general"
	}
}
