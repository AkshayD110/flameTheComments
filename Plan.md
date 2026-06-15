# HN Comment Flamegraph — Plan

## Goal

Build a local web app that turns a Hacker News discussion thread into an interactive “comment flamegraph” for navigating large discussions.

User flow:

```bash
hn-flame 48533848
```

Then browser opens:

```text
http://localhost:3000/item/48533848
```

The page shows an interactive flamegraph of all comments. Clicking a flame block opens the comment text and its surrounding context.

---

# Product Concept

## What we are building

A localhost application for visual exploration of Hacker News comment trees.

It will:

1. Fetch a Hacker News item by `item?id`.
2. Fetch all comments and replies.
3. Build a tree representation.
4. Compute metrics for each comment subtree.
5. Render an interactive flamegraph.
6. Let the user click, search, filter, and read comments.

## Initial target site

Hacker News item pages:

```text
https://news.ycombinator.com/item?id=48533848
```

Input can be either:

```bash
hn-flame 48533848
```

or:

```bash
hn-flame https://news.ycombinator.com/item?id=48533848
```

---

# Recommended Architecture

## Shape

A CLI-launched localhost web app.

```text
CLI command
   ↓
local server
   ↓
HN API / scraping
   ↓
normalized comment tree
   ↓
browser UI
```

## Why this architecture

- Flamegraphs need rich visual interaction.
- Localhost avoids deployment/auth/account complexity.
- CLI makes it easy to run against arbitrary HN threads.
- Later, this can become a browser extension or hosted app if desired.

---

# Proposed Tech Stack

## Decision

Use **Go for the CLI/backend/local server**, with a **React + Vite + D3 frontend** served by Go.

Recommended stack:

- CLI: Go, likely `cobra` later if command complexity grows; standard `flag` is enough for MVP 1.
- Server: Go `net/http`, optionally `chi` if routing grows.
- Fetching: Go `net/http` client with bounded goroutine concurrency.
- Cache: file-based JSON cache for MVP 1; SQLite can be introduced later.
- Frontend: React + Vite + TypeScript.
- Visualization: D3 using SVG initially.
- Packaging: Go single binary with embedded frontend assets using `embed`.

## Why Go

Go is the right fit for the core product because this is primarily a local native-feeling tool:

- excellent CLI and localhost server ergonomics
- easy single-binary distribution
- strong concurrency model for fetching large HN threads
- no Python virtualenv/dependency issues for users
- straightforward cross-platform builds
- easy to embed the compiled web UI into the binary

Python remains attractive for later AI-heavy features, but MVP 1 and MVP 2 are mostly fetching, normalization, metrics, caching, serving, and visualization. Those are very Go-shaped problems.

## Planned project shape

```text
hn-flame/
  cmd/
    hn-flame/
      main.go
  internal/
    browser/
      open.go
    cache/
      cache.go
    hn/
      client.go
      fetcher.go
      normalize.go
      metrics.go
    model/
      types.go
    server/
      server.go
      routes.go
  web/
    package.json
    vite.config.ts
    src/
      App.tsx
      Flamegraph.tsx
      CommentReader.tsx
  webdist/
    embedded build output
```

The frontend can still be TypeScript because D3/browser visualization is best handled in the web layer. Go owns the CLI, fetching, caching, API, and distribution story.

---

# Data Source

## Hacker News Firebase API

HN has a public API:

```text
https://hacker-news.firebaseio.com/v0/item/48533848.json
```

This returns the item and its `kids`.

Each child comment is another item:

```text
https://hacker-news.firebaseio.com/v0/item/{commentId}.json
```

We recursively fetch children.

## Important fields

For story:

```json
{
  "id": 48533848,
  "type": "story",
  "by": "author",
  "time": 1234567890,
  "title": "Story title",
  "url": "...",
  "text": "...",
  "kids": [1, 2, 3],
  "descendants": 1000
}
```

For comment:

```json
{
  "id": 123,
  "type": "comment",
  "by": "commenter",
  "time": 1234567890,
  "text": "comment html",
  "parent": 48533848,
  "kids": [4, 5, 6],
  "deleted": true,
  "dead": true
}
```

---

# Data Model

## Raw HN item

```go
type HNItem struct {
    ID          int    `json:"id"`
    Deleted     bool   `json:"deleted,omitempty"`
    Type        string `json:"type,omitempty"`
    By          string `json:"by,omitempty"`
    Time        int64  `json:"time,omitempty"`
    Text        string `json:"text,omitempty"`
    Dead        bool   `json:"dead,omitempty"`
    Parent      int    `json:"parent,omitempty"`
    Poll        int    `json:"poll,omitempty"`
    Kids        []int  `json:"kids,omitempty"`
    URL         string `json:"url,omitempty"`
    Score       int    `json:"score,omitempty"`
    Title       string `json:"title,omitempty"`
    Descendants int    `json:"descendants,omitempty"`
}
```

## Normalized comment node

```go
type CommentNode struct {
    ID        int            `json:"id"`
    ParentID  *int           `json:"parentId"`
    Author    string         `json:"author,omitempty"`
    Time      int64          `json:"time,omitempty"`
    TextHTML  string         `json:"textHtml"`
    TextPlain string         `json:"textPlain"`
    Deleted   bool           `json:"deleted"`
    Dead      bool           `json:"dead"`
    Depth     int            `json:"depth"`
    Children       []*CommentNode `json:"children"`
    ChildIDs       []int          `json:"childIds,omitempty"`
    ChildrenLoaded bool           `json:"childrenLoaded"`
    Metrics        NodeMetrics    `json:"metrics"`
}

type NodeMetrics struct {
    DirectReplyCount       int `json:"directReplyCount"`
    SubtreeCommentCount    int `json:"subtreeCommentCount"`
    SubtreeWordCount       int `json:"subtreeWordCount"`
    EstimatedReadingSecs   int `json:"estimatedReadingSeconds"`
    MaxDepthBelow          int `json:"maxDepthBelow"`
}
```

## Thread model

```go
type ThreadModel struct {
    ID          int          `json:"id"`
    Title       string       `json:"title"`
    URL         string       `json:"url,omitempty"`
    Author      string       `json:"author,omitempty"`
    Time        int64        `json:"time,omitempty"`
    Score       int          `json:"score,omitempty"`
    Descendants int          `json:"descendants,omitempty"`
    Root        *CommentNode `json:"root"`
    FetchedAt   string       `json:"fetchedAt"`
}
```

The story itself can act as the root node.

---

# Flamegraph Mapping

## Node placement

Each comment is a rectangle.

```text
x position: cumulative order among siblings/subtrees
width: metric value
y position: depth
height: fixed row height
```

## Width metric options

MVP should support one metric first.

Recommended initial metric:

```text
width = subtreeCommentCount
```

This makes large discussion branches visually obvious.

Later options:

```text
width = subtreeWordCount
width = estimatedReadingSeconds
width = directReplyCount
```

## Color options

MVP color:

```text
color = depth
```

Later:

```text
color = author
color = age
color = comment length
color = read/unread
```

## Orientation

Start with **icicle orientation**, not classic bottom-up flamegraph.

Reason: comments naturally read top-down.

The UI can call it “flamegraph style” and later add a toggle:

```text
Top-down icicle mode
Bottom-up flamegraph mode
```

MVP should use top-down.

---

# MVP 1 — Local HN Comment Flamegraph Viewer

## Objective

Build the simplest useful local app:

> Given an HN item id, fetch the discussion and display an interactive flamegraph where users can click a block to read the comment.

## MVP 1 Scope

### CLI

Command:

```bash
hn-flame 48533848
```

Also support:

```bash
hn-flame https://news.ycombinator.com/item?id=48533848
```

Behavior:

1. Parse item id.
2. Start local server.
3. Open browser to `/item/:id`.

Potential dev command:

```bash
go run ./cmd/hn-flame 48533848
```

Frontend-only development can still use Vite from the `web/` directory when working on UI details.

### Backend API

Endpoints:

```text
GET /api/thread/:id/initial
GET /api/thread/:id/subtree/:commentId
GET /api/thread/:id
```

Behavior:

- `/initial` returns the story plus top-level comments quickly, without recursively loading every reply.
- `/subtree/:commentId` loads one full comment subtree on demand.
- `/thread/:id` remains available as a full-tree endpoint for debugging/export and can be fetched in the background after initial render to eventually hydrate the full graph.

Optional:

```text
GET /api/health
```

### Fetcher

Implement progressive HN API fetching.

Features:

- Fetch story item.
- Fetch top-level comments first for immediate rendering.
- Load deeper comment subtrees on demand.
- After initial render, start a background full-thread fetch so the entire graph eventually hydrates even if the user does nothing.
- Preserve HN ordering from `kids`.
- Handle deleted/dead comments.
- Avoid duplicate fetches.
- Limit concurrency to avoid hammering API.
- Keep a full recursive fetch path for cache/export/debug use.

Concurrency example:

```text
max 8 or 16 parallel requests
```

Initial loading should be breadth-first/user-perceived-fast rather than waiting for the entire tree. The page should show something useful after story + top-level comments are available.

### Caching

MVP 1 can use file-based cache.

Example:

```text
.cache/hn/items/48533848.json
.cache/hn/items/123456.json
.cache/hn/threads/48533848.json
```

Cache behavior:

- Cache raw item responses.
- Re-fetch if older than a configurable TTL.
- Default TTL: 1 hour.
- CLI option:

```bash
hn-flame 48533848 --refresh
```

### Frontend Layout

Basic two-pane UI:

```text
+----------------------------------------------------------+
| Header: title, author, score, HN link, fetched time       |
+-------------------------------+--------------------------+
| Flamegraph                    | Comment reader           |
|                               |                          |
| [rect][rect][rect]            | selected comment author  |
|   [rect][rect]                | selected comment time    |
|     [rect]                    | selected comment text    |
|                               |                          |
+-------------------------------+--------------------------+
```

### Flamegraph Interactions

MVP 1 interactions:

- Hover rectangle:
  - show tooltip with author
  - first 200 chars
  - direct replies
  - subtree comments, marked approximate when descendants are not fully loaded
- Click rectangle:
  - select comment
  - show full comment in side panel
  - if the comment has unloaded replies, fetch that subtree in the background and update the graph
- Click selected comment’s parent/children links in side panel.
- Zoom/focus on subtree.
- Button to reset to full thread.
- Visually distinguish fully loaded branches from partially loaded branches.

### Comment Reader

For selected comment show:

- author
- timestamp
- depth
- direct reply count
- subtree comment count
- estimated reading time
- full comment HTML rendered safely
- HN comment link:

```text
https://news.ycombinator.com/item?id={commentId}
```

### Metrics

Compute these for each node:

```text
directReplyCount
subtreeCommentCount
subtreeWordCount
estimatedReadingSeconds
maxDepthBelow
```

Estimated reading time:

```text
words / 220 words per minute
```

### MVP 1 Acceptance Criteria

We are done with MVP 1 when:

1. Running `hn-flame 48533848` opens a local browser page.
2. The app fetches story + top-level comments first and renders without waiting for the entire thread.
3. The page displays a flamegraph/icicle visualization.
4. Large discussion branches are visibly wider, using approximate widths until subtrees are loaded.
5. Clicking a block shows the comment in a side panel.
6. Clicking/focusing a partially loaded block loads its subtree and updates the graph.
7. After initial render, the app continues hydrating the full thread in the background.
8. User can reset from a focused subtree to the full thread.
9. Deleted/dead comments do not crash the app.
10. Threads with 1000+ comments show an initial useful view within a few seconds.

---

# MVP 2 — Reading and Navigation Experience

## Objective

Turn the visualization from a novelty into a practical discussion reader.

MVP 1 answers:

> Where is the discussion?

MVP 2 answers:

> How do I efficiently read the parts I care about?

## MVP 2 Scope

### Search

Add search box.

Search should match:

- comment text
- author
- comment id

Behavior:

- highlight matching blocks in graph
- show matching comments in a result list
- clicking result selects comment

Example:

```text
Search: "rust"
```

Graph highlights all comments mentioning “rust”.

### Author Highlighting

Click author name in reader:

- highlight all comments by that author
- show count
- next/previous comment by same author

Useful for threads where experts participate repeatedly.

### Navigation

Keyboard shortcuts:

```text
j / ArrowDown     next visible comment
k / ArrowUp       previous visible comment
h / ArrowLeft     parent
l / ArrowRight    first child
Enter             focus selected subtree
Escape            reset focus / clear modal
/                 search
```

### Read State

Local read tracking.

Store in browser localStorage or local cache:

```text
readCommentsByThread: {
  "48533848": [commentId1, commentId2]
}
```

Features:

- mark selected comment as read
- optionally mark subtree as read
- visually dim read comments
- show unread count

### Better Width Modes

Add metric selector:

```text
Width by:
- Subtree comments
- Reading time
- Word count
- Direct replies
```

This is important because “most replies” and “most reading effort” are not always the same.

### Sorting Modes

Add top-level branch sorting:

```text
Order:
- HN order
- Largest subtree
- Longest reading time
- Deepest branch
- Newest activity
```

Default remains HN order.

### Minimap / Breadcrumb

When focused on a subtree:

```text
Post > top-level comment by alice > reply by bob
```

Allow jumping back to ancestors.

### Better Tooltips

Tooltip contents:

- author
- age
- first 300 chars
- direct replies
- subtree comments
- reading time
- depth

### Performance Improvements

For 1000–5000 comments:

- Progressive initial render: load story + top-level comments first.
- Lazy subtree loading: fetch replies when a branch is selected/focused.
- Show per-node loading states and `childrenLoaded` markers.
- Use approximate widths for unloaded branches, then recompute once loaded.
- Cache every fetched HN item immediately, including partial fetches.
- Use SVG initially, but consider Canvas if sluggish.
- Do not render labels for tiny blocks.
- Consider hiding or aggregating rectangles narrower than 1px.
- Virtualize comment lists.
- Debounce search.
- Memoize layout calculations.
- Avoid rendering full HTML for every comment until selected.

Later, add Server-Sent Events:

```text
GET /api/thread/:id/stream
```

SSE would let the server fetch breadth-first and push comment nodes to the browser as they arrive, so the graph fills itself in live without polling or waiting for full JSON responses.

### MVP 2 Acceptance Criteria

We are done with MVP 2 when:

1. User can search comment text and authors.
2. Matching comments are highlighted in the graph.
3. User can navigate mostly with keyboard.
4. User can mark comments/subtrees as read.
5. User can change width metric.
6. User can sort top-level branches.
7. App remains responsive on very large HN threads.
8. Reading a 1000-comment thread feels meaningfully easier than using HN directly.

---

# MVP 3 — Insight, Summaries, and Advanced Exploration

## Objective

Make the app not just a visual navigator, but a discussion intelligence tool.

MVP 3 answers:

> What is this giant discussion actually about, and which branches are worth reading?

## MVP 3 Scope

### Subtree Summaries

Add optional local/LLM-powered summaries.

For any comment branch:

```text
Summarize this subtree
```

Output:

- main claim
- key arguments
- counterarguments
- notable links
- sentiment/tone
- “worth reading?” recommendation

Could be opt-in to avoid API cost/privacy issues.

### Branch Labels

Automatically label large branches.

Examples:

```text
"Performance discussion"
"Legal concerns"
"Personal anecdote"
"Rust vs Go debate"
"Security implications"
"Off-topic argument"
```

Display labels on wide enough blocks.

### Semantic Search

Instead of only keyword search:

```text
"comments skeptical of the article"
"technical criticism"
"people discussing pricing"
"security risks"
```

This requires embeddings and a local/vector cache.

Potential local stack:

- SQLite + sqlite-vec
- LanceDB
- local embeddings through Ollama
- remote embeddings as optional config

### Quality Signals

Try to surface likely high-signal branches.

Potential heuristics:

```text
high signal =
  long comments
  many unique authors
  links/code present
  lower back-and-forth repetition
  author reputation maybe unavailable
  high subtree branching
```

Potential low-signal/flamewar signals:

```text
many two-person back-and-forth replies
deep narrow chains
repeated short replies
negative sentiment
```

Represent with visual overlays:

```text
green outline = likely high signal
red outline = likely flamewar
gray = likely low information
```

Important: keep these as “hints”, not judgments.

### Conversation Shape Analysis

Show thread stats:

- total comments
- unique authors
- deepest chain
- largest branch
- most active authors
- median comment length
- top-level branch distribution
- percentage of comments in top 5 branches

This could appear in a side panel:

```text
Thread anatomy
- 1,247 comments
- 389 authors
- deepest chain: 34
- largest branch: 212 comments
- top 5 branches contain 51% of discussion
```

### Export

Allow exporting:

```text
Export as:
- standalone HTML
- SVG
- PNG
- JSON
- Markdown summary
```

Standalone HTML is especially useful:

```bash
hn-flame export 48533848 --format html
```

### Browser Extension Prototype

Optional MVP 3 extension:

On HN item pages, inject button:

```text
View Flamegraph
```

Clicking it opens:

```text
http://localhost:3000/item/48533848
```

Or injects inline graph directly into HN.

Do not do this before MVP 3.

### Multi-site Support

Generalize beyond HN.

Potential future targets:

- Reddit
- Lobsters
- GitHub issues
- GitHub PR reviews
- YouTube comments
- Slack/Discord exports
- mailing list archives

To support this, introduce adapter interface:

```go
type ThreadSourceAdapter interface {
    ParseInput(input string) (ThreadRef, error)
    FetchThread(ctx context.Context, ref ThreadRef) (*ThreadModel, error)
}
```

HN is the first adapter.

### MVP 3 Acceptance Criteria

We are done with MVP 3 when:

1. User can generate summaries for large branches.
2. Large branches can be labeled by topic.
3. Search supports semantic intent, not just keywords.
4. App highlights potentially valuable branches.
5. User can export the visualization.
6. Architecture supports adding non-HN discussion sources.
7. The app feels like an analysis tool, not just a viewer.

---

# Overall Build Order

## Phase 0 — Prototype Spike

Before MVP 1, do a tiny technical spike:

1. Fetch one HN thread.
2. Build recursive tree.
3. Compute subtree counts.
4. Render static SVG icicle chart.

No polish. Just validate the idea.

## Phase 1 — MVP 1

Build the usable localhost viewer.

Priority order:

1. HN fetcher
2. tree normalization
3. metrics
4. local server
5. basic frontend
6. flamegraph rendering
7. click-to-read
8. cache
9. CLI wrapper

## Phase 2 — MVP 2

Improve reading workflow.

Priority order:

1. search
2. keyboard navigation
3. read state
4. width mode selector
5. sorting modes
6. performance polish

## Phase 3 — MVP 3

Add intelligence.

Priority order:

1. thread stats
2. export
3. subtree summary
4. branch labels
5. semantic search
6. multi-site adapter abstraction

---

# Suggested Initial UI

```text
HN Flamegraph

Title: Example HN Story
Comments: 1,247
Width: [Subtree comments v]
Order: [HN order v]
Search: [________________]

+-------------------------------------------------------------+
|                                                             |
|                   Flamegraph / Icicle View                  |
|                                                             |
+-------------------------------------------------------------+

Selected Comment
---------------------------------------------------------------
author · 2 hours ago · depth 3 · 42 comments below

Comment text...
---------------------------------------------------------------
[View on HN] [Focus subtree] [Mark subtree read]
```

On wider screens:

```text
+-----------------------------------------+-------------------+
| Flamegraph                              | Comment Reader    |
|                                         |                   |
|                                         |                   |
+-----------------------------------------+-------------------+
```

---

# Key Product Decisions

## Decision 1: Top-down or bottom-up?

Start with **top-down icicle**.

Later add bottom-up flamegraph toggle.

## Decision 2: Width metric?

Start with:

```text
subtreeCommentCount
```

Add reading-time width in MVP 2.

## Decision 3: HN API or scraping?

Use the HN Firebase API.

Avoid scraping unless the API lacks something critical.

## Decision 4: Localhost or hosted?

Localhost first.

Hosted version later only if sharing/collaboration matters.

## Decision 5: SVG or Canvas?

Start with SVG.

Move to Canvas only if large threads are too slow.

---

# Final Recommendation

Build this as:

```text
Go CLI + Go local server + embedded React/Vite frontend + D3 visualization
```

MVP 1 should focus on one magical experience:

```bash
hn-flame 48533848
```

Browser opens, and within seconds you see the full discussion shape as a navigable flamegraph.
