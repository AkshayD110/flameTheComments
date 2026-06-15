export type NodeMetrics = {
  directReplyCount: number;
  subtreeCommentCount: number;
  subtreeWordCount: number;
  estimatedReadingSeconds: number;
  maxDepthBelow: number;
};

export type CommentNode = {
  id: number;
  parentId: number | null;
  author?: string;
  time?: number;
  textHtml: string;
  textPlain: string;
  deleted: boolean;
  dead: boolean;
  depth: number;
  children: CommentNode[];
  childIds?: number[];
  childrenLoaded: boolean;
  metrics: NodeMetrics;
};

export type ThreadModel = {
  id: number;
  title: string;
  url?: string;
  author?: string;
  time?: number;
  score?: number;
  descendants?: number;
  root: CommentNode;
  fetchedAt: string;
};
