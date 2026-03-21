package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const maxWorkers = 20

func main() {
	// CLI flags
	limit := flag.Int("limit", 30, "max number of topics to process")
	category := flag.String("category", "", "filter by category name")
	keyword := flag.String("keyword", "", "filter by keyword in title/content/author")
	since := flag.String("since", "", "only topics from the last duration (e.g., 24h, 72h, 168h)")
	tags := flag.String("tags", "", "comma-separated tags to filter by")
	deep := flag.Bool("deep", false, "fetch individual topic pages for richer content")
	noLLM := flag.Bool("no-llm", false, "use rule-based summary instead of Claude Sonnet 4.6 (no API key needed)")
	output := flag.String("output", "", "write output to file (default: stdout)")
	feedURL := flag.String("feed-url", defaultFeedURL, "RSS feed URL")
	workers := flag.Int("workers", 5, "number of concurrent scrapers for --deep mode")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "delphgo — Chief Delphi activity reader and summarizer\n\n")
		fmt.Fprintf(os.Stderr, "Usage: delphgo [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Requires ANTHROPIC_API_KEY for AI summaries (default). Use --no-llm to skip.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  delphgo                          # AI summary of latest topics\n")
		fmt.Fprintf(os.Stderr, "  delphgo --deep                   # include reply content in AI summary\n")
		fmt.Fprintf(os.Stderr, "  delphgo --category Technical     # filter to Technical category\n")
		fmt.Fprintf(os.Stderr, "  delphgo --keyword swerve --deep  # search + deep scrape\n")
		fmt.Fprintf(os.Stderr, "  delphgo --since 24h              # last 24 hours only\n")
		fmt.Fprintf(os.Stderr, "  delphgo --no-llm --output out.md # rule-based summary, no API key needed\n")
	}
	flag.Parse()

	// Fetch RSS feed
	fmt.Fprintf(os.Stderr, "Fetching feed from %s ...\n", *feedURL)
	topics, err := FetchFeed(*feedURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Fetched %d topics from RSS feed.\n", len(topics))

	// Apply filters
	var sinceDur time.Duration
	if *since != "" {
		sinceDur, err = time.ParseDuration(*since)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid --since duration %q: %v\n", *since, err)
			os.Exit(1)
		}
	}

	var tagList []string
	if *tags != "" {
		for _, tag := range strings.Split(*tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagList = append(tagList, tag)
			}
		}
	}

	topics = FilterTopics(topics, FilterOptions{
		Category: *category,
		Keyword:  *keyword,
		Tags:     tagList,
		Since:    sinceDur,
	})
	fmt.Fprintf(os.Stderr, "%d topics after filtering.\n", len(topics))

	// Apply limit
	if *limit > 0 && len(topics) > *limit {
		topics = topics[:*limit]
	}

	// Deep scrape if requested
	if *deep && len(topics) > 0 {
		if *workers > maxWorkers {
			*workers = maxWorkers
		}
		fmt.Fprintf(os.Stderr, "Deep scraping %d topics (%d workers)...\n", len(topics), *workers)
		topics = deepScrape(topics, *workers)
	}

	// Generate summary
	var md string
	if *noLLM {
		md = GenerateSummary(topics)
	} else {
		fmt.Fprintf(os.Stderr, "Generating AI summary with Claude Sonnet 4.6...\n")
		md, err = LLMSummary(topics)
		if err != nil {
			fmt.Fprintf(os.Stderr, "LLM summary failed (%v), falling back to rule-based summary.\n", err)
			md = GenerateSummary(topics)
		}
	}

	// Output
	if *output != "" {
		if err := os.WriteFile(*output, []byte(md), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Summary written to %s\n", *output)
	} else {
		fmt.Print(md)
	}
}

func deepScrape(topics []Topic, numWorkers int) []Topic {
	type result struct {
		index int
		posts []Post
		rc    int
		vc    int
		lc    int
	}

	ch := make(chan int, len(topics))
	results := make(chan result, len(topics))

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range ch {
				if topics[idx].Link == "" {
					results <- result{index: idx}
					continue
				}
				posts, rc, vc, lc, err := ScrapeTopic(topics[idx].Link)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  warning: scrape failed for %q: %v\n", topics[idx].Title, err)
					results <- result{index: idx}
				} else {
					results <- result{index: idx, posts: posts, rc: rc, vc: vc, lc: lc}
				}
				time.Sleep(200 * time.Millisecond) // polite rate limiting
			}
		}()
	}

	for i := range topics {
		ch <- i
	}
	close(ch)

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		topics[r.index].Posts = r.posts
		topics[r.index].ReplyCount = r.rc
		topics[r.index].ViewCount = r.vc
		topics[r.index].LikeCount = r.lc
		// Use post count as reply proxy when Discourse doesn't render stats server-side
		if topics[r.index].ReplyCount == 0 && len(r.posts) > 1 {
			topics[r.index].ReplyCount = len(r.posts) - 1
		}
	}

	scraped := 0
	for _, t := range topics {
		if t.ReplyCount > 0 || len(t.Posts) > 0 {
			scraped++
		}
	}
	fmt.Fprintf(os.Stderr, "Deep scrape complete: %d/%d topics enriched.\n", scraped, len(topics))

	return topics
}
