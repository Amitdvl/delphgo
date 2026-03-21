package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

const summaryModel = anthropic.ModelClaudeSonnet4_6

// LLMSummary calls Claude Sonnet 4.6 to generate a high-quality Markdown
// summary of the provided topics. It streams the response and returns the
// complete text when done.
func LLMSummary(topics []Topic) (string, error) {
	if len(topics) == 0 {
		return "# Chief Delphi Summary\n\nNo topics found matching the given criteria.\n", nil
	}

	client := anthropic.NewClient() // reads ANTHROPIC_API_KEY from env

	prompt := buildPrompt(topics)

	stream := client.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     summaryModel,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{Text: "You are an expert summarizer of robotics engineering discussions. " +
				"You read Chief Delphi forum activity and produce concise, high-signal Markdown summaries. " +
				"Be direct and specific. Avoid filler phrases. Use real usernames, topic titles, and links from the data provided."},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	message := anthropic.Message{}
	for stream.Next() {
		message.Accumulate(stream.Current())
	}
	if err := stream.Err(); err != nil {
		return "", fmt.Errorf("LLM stream error: %w", err)
	}

	var sb strings.Builder
	for _, block := range message.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(b.Text)
		}
	}
	return sb.String(), nil
}

func buildPrompt(topics []Topic) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Today is %s. Here are %d recent Chief Delphi forum topics. ",
		time.Now().Format("2006-01-02"), len(topics)))
	sb.WriteString("Please write a Markdown summary with these exact sections:\n\n")
	sb.WriteString("1. **Main Themes** — What subjects dominate the discussion? Group by theme, not just category.\n")
	sb.WriteString("2. **Important Threads** — The most notable or high-engagement threads. Include author, link, and a 1-2 sentence description.\n")
	sb.WriteString("3. **Actionable Takeaways** — Things teams should know or act on: help requests, announcements, rule clarifications, events.\n")
	sb.WriteString("4. **Useful Links** — Curated list of links with brief context labels.\n")
	sb.WriteString("5. **Repeated Opinions & Sentiment** — What recurring views, concerns, or enthusiasm is the community expressing?\n\n")
	sb.WriteString("---\n\n")
	sb.WriteString("TOPIC DATA:\n\n")

	for i, t := range topics {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, t.Title))
		sb.WriteString(fmt.Sprintf("- **Link:** %s\n", t.Link))
		sb.WriteString(fmt.Sprintf("- **Author:** %s\n", authorOrUnknown(t.Author)))
		sb.WriteString(fmt.Sprintf("- **Category:** %s\n", categoryOrUncategorized(t.Category)))
		if !t.PublishedAt.IsZero() {
			sb.WriteString(fmt.Sprintf("- **Posted:** %s\n", t.PublishedAt.Format("Jan 2, 2006 15:04")))
		}
		if t.ReplyCount > 0 {
			sb.WriteString(fmt.Sprintf("- **Replies:** %d\n", t.ReplyCount))
		}
		if t.ViewCount > 0 {
			sb.WriteString(fmt.Sprintf("- **Views:** %d\n", t.ViewCount))
		}
		if len(t.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("- **Tags:** %s\n", strings.Join(t.Tags, ", ")))
		}

		// First post excerpt from RSS description
		excerpt := firstSentences(stripHTML(t.Description), 3)
		if excerpt != "" {
			sb.WriteString(fmt.Sprintf("- **Excerpt:** %s\n", excerpt))
		}

		// Notable replies from deep scrape
		if len(t.Posts) > 1 {
			sb.WriteString("- **Notable replies:**\n")
			for j, p := range t.Posts[1:] {
				if j >= 3 {
					break
				}
				snippet := firstSentences(p.Content, 1)
				if snippet == "" {
					continue
				}
				sb.WriteString(fmt.Sprintf("  - %s: %s\n", authorOrUnknown(p.Author), snippet))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func categoryOrUncategorized(c string) string {
	if c == "" {
		return "Uncategorized"
	}
	return c
}
