# Delphgo
<img width="1536" height="1024" alt="delphgo" src="https://github.com/user-attachments/assets/ef232956-b7dd-40d7-8d59-e814388c0d09" />

A read-only Go CLI that fetches and summarizes [Chief Delphi](https://www.chiefdelphi.com) forum activity. Uses the public RSS feed as the primary ingestion path, with optional deep scraping of individual topic pages for richer content.

## Install

```bash
git clone https://github.com/Amitdvl/delphgo.git
cd delphgo
go build -o delphgo .
```

Requires Go 1.21+.

## Usage

```
delphgo [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--limit N` | `30` | Max number of topics to process |
| `--category NAME` | | Filter by category name (e.g. `Technical`, `Programming`, `Robot Showcase`) |
| `--keyword WORD` | | Filter by keyword in title, content, or author |
| `--tags TAG1,TAG2` | | Filter by comma-separated tags |
| `--since DURATION` | | Only topics from the last duration (e.g. `24h`, `72h`, `168h`) |
| `--deep` | `false` | Fetch individual topic pages to extract replies and engagement data |
| `--workers N` | `5` | Number of concurrent scrapers in `--deep` mode |
| `--output FILE` | stdout | Write Markdown output to a file |
| `--feed-url URL` | CD latest RSS | Override the RSS feed URL |

### Examples

```bash
# Summarize the latest 30 topics
./delphgo

# Last 24 hours only
./delphgo --since 24h

# Filter to a specific category
./delphgo --category Programming

# Search for a keyword across titles and content
./delphgo --keyword swerve

# Deep scrape: fetch replies and show engagement ranking
./delphgo --deep --limit 10

# Combine filters with deep scrape and write to file
./delphgo --category Technical --since 72h --deep --output summary.md

# Use a category-specific RSS feed
./delphgo --feed-url https://www.chiefdelphi.com/c/technical/43.rss --deep
```

## Output

`delphgo` generates a Markdown document with five sections:

**Main Themes** — Topics grouped by category with author, timestamp, and engagement counts.

**Important Threads** — Topics ranked by reply count and views. In `--deep` mode, includes quoted snippets from notable replies with attribution.

**Actionable Takeaways** — Topics flagged as help requests, announcements, rule changes, events, or troubleshooting threads.

**Useful Links** — Full list of topic links with category labels.

**Repeated Opinions & Sentiment** — Most frequent terms across topic titles, revealing recurring themes in the community.

Progress and warnings are written to stderr; the Markdown summary goes to stdout (or `--output`).

## How it works

1. Fetches the Chief Delphi RSS feed (`/latest.rss` by default)
2. Parses the XML to extract topic title, author, category, timestamp, and first-post excerpt
3. Applies any active filters (category, keyword, tags, time window)
4. If `--deep` is set, concurrently fetches each topic page and parses Discourse's crawler-friendly HTML view to extract reply authors and content
5. Generates a structured Markdown summary

No login, API keys, or credentials required — everything comes from public pages.

## License

MIT
