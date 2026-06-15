# HN Flamegraph

HN Flamegraph is a local web app for exploring large Hacker News discussion threads as an interactive flamegraph-style visualization.

It fetches an HN item and its comments, builds the comment tree, and renders each comment as a block. Wider blocks represent larger discussion subtrees. Clicking a block shows the selected comment in a reader panel and lets you focus on that branch.

## What it does

- Opens a localhost UI for a Hacker News item.
- Fetches comments from the Hacker News Firebase API.
- Shows the discussion as a top-down flamegraph/icicle view.
- Loads the initial thread view quickly, then continues hydrating the full thread.
- Lets you click comments, inspect replies, and focus/reset subtrees.
- Caches fetched HN data locally.

## How to use

Install frontend dependencies and build the embedded UI:

```bash
cd web
npm install
npm run build
cd ..
```

Run against an HN item id:

```bash
go run ./cmd/hn-flame 48533848
```

Or run against an HN item URL:

```bash
go run ./cmd/hn-flame https://news.ycombinator.com/item?id=48533848
```

Build a binary:

```bash
go build -o hn-flame ./cmd/hn-flame
./hn-flame 48533848
```

Useful flags:

```bash
--port 3000       localhost port
--refresh         bypass cached HN items
--no-open         do not open browser automatically
--cache-dir DIR   cache directory
```
