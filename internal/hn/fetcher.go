package hn

import (
	"context"
	"html"
	"regexp"
	"strings"
	"sync"
	"time"

	"hn-flame/internal/model"
)

var tagRE = regexp.MustCompile(`<[^>]+>`)

func FetchInitialThread(ctx context.Context, client *Client, id int) (*model.ThreadModel, error) {
	rootItem, err := client.FetchItem(ctx, id)
	if err != nil {
		return nil, err
	}
	root := itemToNode(rootItem, nil, 0)
	root.TextHTML = rootItem.Text
	root.TextPlain = plainText(rootItem.Text)
	root.ChildIDs = append([]int(nil), rootItem.Kids...)
	root.ChildrenLoaded = true
	root.Children = fetchChildStubs(ctx, client, rootItem.Kids, rootItem.ID, 1)
	computeMetrics(root, true)

	return threadFromRoot(rootItem, root), nil
}

func FetchThread(ctx context.Context, client *Client, id int) (*model.ThreadModel, error) {
	rootItem, err := client.FetchItem(ctx, id)
	if err != nil {
		return nil, err
	}
	root := itemToNode(rootItem, nil, 0)
	root.TextHTML = rootItem.Text
	root.TextPlain = plainText(rootItem.Text)
	root.ChildIDs = append([]int(nil), rootItem.Kids...)
	root.Children = fetchChildren(ctx, client, rootItem.Kids, rootItem.ID, 1)
	root.ChildrenLoaded = true
	computeMetrics(root, true)

	thread := threadFromRoot(rootItem, root)
	if client.Cache != nil {
		_ = client.Cache.WriteJSON("threads", id, thread)
	}
	return thread, nil
}

func FetchSubtree(ctx context.Context, client *Client, id int) (*model.CommentNode, error) {
	item, err := client.FetchItem(ctx, id)
	if err != nil {
		return nil, err
	}
	parent := item.Parent
	node := itemToNode(item, &parent, 0)
	node.Children = fetchChildren(ctx, client, item.Kids, item.ID, 1)
	node.ChildrenLoaded = true
	computeMetrics(node, false)
	return node, nil
}

func threadFromRoot(item *model.HNItem, root *model.CommentNode) *model.ThreadModel {
	return &model.ThreadModel{
		ID:          item.ID,
		Title:       item.Title,
		URL:         item.URL,
		Author:      item.By,
		Time:        item.Time,
		Score:       item.Score,
		Descendants: item.Descendants,
		Root:        root,
		FetchedAt:   time.Now().Format(time.RFC3339),
	}
}

func fetchChildStubs(ctx context.Context, client *Client, ids []int, parent int, depth int) []*model.CommentNode {
	children := make([]*model.CommentNode, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		i, id := i, id
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, err := client.FetchItem(ctx, id)
			if err != nil || item == nil {
				children[i] = &model.CommentNode{ID: id, ParentID: &parent, Deleted: true, Depth: depth, TextPlain: "[failed to load]", ChildrenLoaded: true}
				return
			}
			node := itemToNode(item, &parent, depth)
			node.ChildrenLoaded = len(item.Kids) == 0
			children[i] = node
		}()
	}
	wg.Wait()
	return children
}

func fetchChildren(ctx context.Context, client *Client, ids []int, parent int, depth int) []*model.CommentNode {
	children := make([]*model.CommentNode, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		i, id := i, id
		wg.Add(1)
		go func() {
			defer wg.Done()
			item, err := client.FetchItem(ctx, id)
			if err != nil || item == nil {
				children[i] = &model.CommentNode{ID: id, ParentID: &parent, Deleted: true, Depth: depth, TextPlain: "[failed to load]", ChildrenLoaded: true}
				return
			}
			node := itemToNode(item, &parent, depth)
			node.Children = fetchChildren(ctx, client, item.Kids, item.ID, depth+1)
			node.ChildrenLoaded = true
			children[i] = node
		}()
	}
	wg.Wait()
	return children
}

func itemToNode(item *model.HNItem, parent *int, depth int) *model.CommentNode {
	text := item.Text
	if item.Deleted {
		text = "[deleted]"
	} else if item.Dead {
		text = "[dead]"
	}
	return &model.CommentNode{
		ID:             item.ID,
		ParentID:       parent,
		Author:         item.By,
		Time:           item.Time,
		TextHTML:       text,
		TextPlain:      plainText(text),
		Deleted:        item.Deleted,
		Dead:           item.Dead,
		Depth:          depth,
		Children:       []*model.CommentNode{},
		ChildIDs:       append([]int(nil), item.Kids...),
		ChildrenLoaded: len(item.Kids) == 0,
	}
}

func plainText(s string) string {
	s = strings.ReplaceAll(s, "<p>", "\n\n")
	s = strings.ReplaceAll(s, "<pre><code>", "\n")
	s = strings.ReplaceAll(s, "</code></pre>", "\n")
	s = tagRE.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

func computeMetrics(node *model.CommentNode, root bool) model.NodeMetrics {
	words := len(strings.Fields(node.TextPlain))
	commentCount := 0
	if !root {
		commentCount = 1
	}
	maxDepth := 0
	for _, child := range node.Children {
		m := computeMetrics(child, false)
		commentCount += m.SubtreeCommentCount
		words += m.SubtreeWordCount
		if m.MaxDepthBelow+1 > maxDepth {
			maxDepth = m.MaxDepthBelow + 1
		}
	}
	if !node.ChildrenLoaded && len(node.ChildIDs) > len(node.Children) {
		// Approximate unloaded descendants so partially loaded branches still have
		// meaningful width in the initial flamegraph.
		commentCount += len(node.ChildIDs) - len(node.Children)
		if maxDepth == 0 {
			maxDepth = 1
		}
	}
	directReplies := len(node.Children)
	if len(node.ChildIDs) > directReplies {
		directReplies = len(node.ChildIDs)
	}
	node.Metrics = model.NodeMetrics{
		DirectReplyCount:     directReplies,
		SubtreeCommentCount:  commentCount,
		SubtreeWordCount:     words,
		EstimatedReadingSecs: (words * 60) / 220,
		MaxDepthBelow:        maxDepth,
	}
	return node.Metrics
}
