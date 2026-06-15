# HN Flamegraph

Localhost Hacker News comment flamegraph viewer.

## Run

```bash
cd web
npm install
npm run build
cd ..
go run ./cmd/hn-flame 48533848
```

Or build a binary:

```bash
go build -o hn-flame ./cmd/hn-flame
./hn-flame https://news.ycombinator.com/item?id=48533848
```

Useful flags:

```bash
--port 3000       localhost port
--refresh         bypass cached HN items
--no-open         do not open browser automatically
--cache-dir DIR   cache directory
```

## MVP 1 features

- Fetches HN item/comment trees from the Firebase API.
- Loads story + top-level comments first for fast initial render.
- Continues loading the full thread in the background after initial render.
- Lazily hydrates deeper subtrees immediately when a branch is selected or focused.
- Caches raw items and normalized threads locally.
- Serves a local web UI from the Go binary.
- Renders an interactive top-down flamegraph/icicle view.
- Width is based on subtree comment count, approximate until a branch is fully loaded.
- Click a block to inspect the comment in a reader panel.
- Focus/reset a subtree.
# flameTheComments
