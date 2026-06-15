package model

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

type NodeMetrics struct {
	DirectReplyCount     int `json:"directReplyCount"`
	SubtreeCommentCount  int `json:"subtreeCommentCount"`
	SubtreeWordCount     int `json:"subtreeWordCount"`
	EstimatedReadingSecs int `json:"estimatedReadingSeconds"`
	MaxDepthBelow        int `json:"maxDepthBelow"`
}

type CommentNode struct {
	ID             int            `json:"id"`
	ParentID       *int           `json:"parentId"`
	Author         string         `json:"author,omitempty"`
	Time           int64          `json:"time,omitempty"`
	TextHTML       string         `json:"textHtml"`
	TextPlain      string         `json:"textPlain"`
	Deleted        bool           `json:"deleted"`
	Dead           bool           `json:"dead"`
	Depth          int            `json:"depth"`
	Children       []*CommentNode `json:"children"`
	ChildIDs       []int          `json:"childIds,omitempty"`
	ChildrenLoaded bool           `json:"childrenLoaded"`
	Metrics        NodeMetrics    `json:"metrics"`
}

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
