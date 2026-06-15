import React, { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import type { CommentNode, ThreadModel } from './types';
import './styles.css';

type Rect = { node: CommentNode; x: number; y: number; w: number; h: number };

function itemIdFromPath(): string {
  const m = window.location.pathname.match(/\/item\/(\d+)/);
  return m?.[1] ?? '';
}

function App() {
  const itemId = itemIdFromPath();
  const [thread, setThread] = useState<ThreadModel | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [focusId, setFocusId] = useState<number | null>(null);
  const [loadingSubtrees, setLoadingSubtrees] = useState<Set<number>>(() => new Set());
  const [loadingFullThread, setLoadingFullThread] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/thread/${itemId}/initial`)
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text());
        return r.json();
      })
      .then((t: ThreadModel) => {
        setThread(t);
        setSelectedId(t.root.id);
        void loadFullThreadInBackground();
      })
      .catch((e) => setError(String(e.message ?? e)))
      .finally(() => setLoading(false));
  }, [itemId]);

  const nodeMap = useMemo(() => {
    const m = new Map<number, CommentNode>();
    if (!thread) return m;
    walk(thread.root, (n) => m.set(n.id, n));
    return m;
  }, [thread]);

  const focusNode = focusId ? nodeMap.get(focusId) : thread?.root;
  const selected = selectedId ? nodeMap.get(selectedId) : thread?.root;

  async function loadFullThreadInBackground() {
    setLoadingFullThread(true);
    try {
      const r = await fetch(`/api/thread/${itemId}`);
      if (!r.ok) throw new Error(await r.text());
      const fullThread: ThreadModel = await r.json();
      setThread(fullThread);
    } catch (e) {
      console.warn('full thread background load failed', e);
    } finally {
      setLoadingFullThread(false);
    }
  }

  function selectNode(id: number) {
    setSelectedId(id);
    const node = nodeMap.get(id);
    if (node && !node.childrenLoaded) void loadSubtree(id);
  }

  async function loadSubtree(id: number) {
    const existing = nodeMap.get(id);
    if (!existing || existing.childrenLoaded || loadingSubtrees.has(id)) return;
    setLoadingSubtrees((prev) => new Set(prev).add(id));
    try {
      const r = await fetch(`/api/thread/${itemId}/subtree/${id}`);
      if (!r.ok) throw new Error(await r.text());
      const subtree: CommentNode = await r.json();
      const adjusted = adjustDepth(subtree, existing.depth);
      setThread((prev) => prev ? { ...prev, root: replaceNode(prev.root, adjusted) } : prev);
    } catch (e) {
      setError(String(e instanceof Error ? e.message : e));
    } finally {
      setLoadingSubtrees((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  }

  if (!itemId) return <Shell><div className="error">Open /item/&lt;hn-id&gt;.</div></Shell>;
  if (loading) return <Shell><div className="loading">Fetching HN thread {itemId}… Large threads can take a few seconds.</div></Shell>;
  if (error) return <Shell><div className="error">{error}</div></Shell>;
  if (!thread || !focusNode) return <Shell><div className="error">No thread loaded.</div></Shell>;

  return (
    <Shell>
      <header className="topbar">
        <div>
          <h1>{thread.title || `HN item ${thread.id}`}</h1>
          <div className="meta">
            by {thread.author || 'unknown'} · {thread.descendants ?? thread.root.metrics.subtreeCommentCount} comments · score {thread.score ?? 0} · fetched {new Date(thread.fetchedAt).toLocaleString()}
          </div>
        </div>
        <div className="links">
          {thread.url && <a href={thread.url} target="_blank">story</a>}
          <a href={`https://news.ycombinator.com/item?id=${thread.id}`} target="_blank">HN</a>
        </div>
      </header>
      <main className="layout">
        <section className="graphPanel">
          <div className="toolbar">
            <strong>Width:</strong> subtree comments
            {loadingFullThread && <span className="status">loading full thread in background…</span>}
            {focusId && <button onClick={() => setFocusId(null)}>Reset focus</button>}
          </div>
          <Flamegraph root={focusNode} selectedId={selectedId} loadingIds={loadingSubtrees} onSelect={selectNode} />
        </section>
        <CommentReader
          node={selected}
          nodeMap={nodeMap}
          loadingIds={loadingSubtrees}
          onSelect={selectNode}
          onLoadSubtree={(id) => void loadSubtree(id)}
          onFocus={(id) => {
            setFocusId(id);
            void loadSubtree(id);
          }}
        />
      </main>
    </Shell>
  );
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="app">{children}</div>;
}

function Flamegraph({ root, selectedId, loadingIds, onSelect }: { root: CommentNode; selectedId: number | null; loadingIds: Set<number>; onSelect: (id: number) => void }) {
  const rowH = 28;
  const gap = 2;
  const width = 1200;
  const maxDepth = root.metrics.maxDepthBelow + 1;
  const height = Math.max(160, (maxDepth + 1) * rowH);
  const rects = useMemo(() => layout(root, 0, 0, width, rowH, gap), [root]);

  return (
    <div className="svgWrap">
      <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="HN comment flamegraph">
        {rects.map((r) => {
          const label = r.w > 80 ? `${r.node.author || 'anon'} · ${r.node.metrics.subtreeCommentCount}` : '';
          return (
            <g key={r.node.id} onClick={() => onSelect(r.node.id)} className="block">
              <title>{tooltip(r.node)}</title>
              <rect
                x={r.x}
                y={r.y}
                width={Math.max(0, r.w - gap)}
                height={r.h - gap}
                rx="3"
                fill={color(r.node.depth)}
                opacity={r.node.childrenLoaded ? 1 : 0.62}
                stroke={r.node.id === selectedId ? '#111827' : r.node.childrenLoaded ? 'rgba(255,255,255,.55)' : '#334155'}
                strokeDasharray={r.node.childrenLoaded ? undefined : '4 2'}
                strokeWidth={r.node.id === selectedId ? 3 : 1}
              />
              {loadingIds.has(r.node.id) && <text x={r.x + 6} y={r.y + 18}>loading…</text>}
              {label && <text x={r.x + 6} y={r.y + 18}>{label}</text>}
            </g>
          );
        })}
      </svg>
    </div>
  );
}

function CommentReader({ node, nodeMap, loadingIds, onSelect, onFocus, onLoadSubtree }: { node?: CommentNode; nodeMap: Map<number, CommentNode>; loadingIds: Set<number>; onSelect: (id: number) => void; onFocus: (id: number) => void; onLoadSubtree: (id: number) => void }) {
  if (!node) return <aside className="reader">Select a block.</aside>;
  const parent = node.parentId ? nodeMap.get(node.parentId) : undefined;
  return (
    <aside className="reader">
      <h2>{node.id === [...nodeMap.values()][0]?.id ? 'Story root' : 'Selected comment'}</h2>
      <div className="meta big">
        {node.author || 'unknown'} · {node.time ? new Date(node.time * 1000).toLocaleString() : 'unknown time'} · depth {node.depth}
      </div>
      <div className="stats">
        <span>{node.metrics.directReplyCount} replies</span>
        <span>{node.metrics.subtreeCommentCount} comments below</span>
        <span>{Math.max(1, Math.round(node.metrics.estimatedReadingSeconds / 60))} min read subtree</span>
      </div>
      <div className="actions">
        <a href={`https://news.ycombinator.com/item?id=${node.id}`} target="_blank">View on HN</a>
        <button onClick={() => onFocus(node.id)}>Focus subtree</button>
        {!node.childrenLoaded && <button onClick={() => onLoadSubtree(node.id)} disabled={loadingIds.has(node.id)}>{loadingIds.has(node.id) ? 'Loading replies…' : `Load ${node.metrics.directReplyCount} replies`}</button>}
        {parent && <button onClick={() => onSelect(parent.id)}>Parent</button>}
      </div>
      <article className="commentText">{node.textPlain || (node.deleted ? '[deleted]' : '[no text]')}</article>
      {!node.childrenLoaded && <div className="notice">Replies are not loaded yet. Click “Load replies” or select/focus this branch to hydrate the subtree.</div>}
      {node.children.length > 0 && (
        <div className="children">
          <strong>Children</strong>
          {node.children.slice(0, 25).map((c) => <button key={c.id} onClick={() => onSelect(c.id)}>{c.author || 'anon'} · {c.metrics.subtreeCommentCount}</button>)}
          {node.children.length > 25 && <span>+ {node.children.length - 25} more</span>}
        </div>
      )}
    </aside>
  );
}

function layout(node: CommentNode, x: number, y: number, w: number, rowH: number, gap: number): Rect[] {
  const rects: Rect[] = [{ node, x, y, w, h: rowH }];
  if (node.children.length === 0) return rects;
  const total = node.children.reduce((sum, c) => sum + weight(c), 0);
  let cursor = x;
  for (const child of node.children) {
    const cw = total > 0 ? w * (weight(child) / total) : w / node.children.length;
    rects.push(...layout(child, cursor, y + rowH, cw, rowH, gap));
    cursor += cw;
  }
  return rects;
}

function weight(n: CommentNode): number {
  return Math.max(1, n.metrics.subtreeCommentCount);
}

function walk(n: CommentNode, fn: (n: CommentNode) => void) {
  fn(n);
  n.children.forEach((c) => walk(c, fn));
}

function replaceNode(current: CommentNode, replacement: CommentNode): CommentNode {
  if (current.id === replacement.id) return replacement;
  return { ...current, children: current.children.map((child) => replaceNode(child, replacement)) };
}

function adjustDepth(node: CommentNode, depth: number): CommentNode {
  return { ...node, depth, children: node.children.map((child) => adjustDepth(child, depth + 1)) };
}

function tooltip(n: CommentNode): string {
  const preview = n.textPlain.length > 220 ? n.textPlain.slice(0, 220) + '…' : n.textPlain;
  const loaded = n.childrenLoaded ? 'loaded' : 'partially loaded';
  return `${n.author || 'unknown'} · ${loaded}\n${n.metrics.directReplyCount} direct replies · ${n.metrics.subtreeCommentCount} subtree comments${n.childrenLoaded ? '' : ' (approx)'}\n${preview}`;
}

function color(depth: number): string {
  const palette = ['#f97316', '#fb923c', '#fdba74', '#facc15', '#a3e635', '#34d399', '#22d3ee', '#60a5fa', '#a78bfa', '#f472b6'];
  return palette[depth % palette.length];
}

createRoot(document.getElementById('root')!).render(<App />);
