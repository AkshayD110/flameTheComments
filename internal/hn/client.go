package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hn-flame/internal/cache"
	"hn-flame/internal/model"
)

const baseURL = "https://hacker-news.firebaseio.com/v0/item/%d.json"

type Client struct {
	HTTP    *http.Client
	Cache   *cache.FileCache
	Refresh bool
	sem     chan struct{}
}

func NewClient(c *cache.FileCache, refresh bool, concurrency int) *Client {
	if concurrency <= 0 {
		concurrency = 12
	}
	return &Client{
		HTTP:    &http.Client{Timeout: 20 * time.Second},
		Cache:   c,
		Refresh: refresh,
		sem:     make(chan struct{}, concurrency),
	}
}

func (c *Client) FetchItem(ctx context.Context, id int) (*model.HNItem, error) {
	var cached model.HNItem
	if c.Cache != nil && c.Cache.ReadJSON("items", id, &cached, c.Refresh) {
		return &cached, nil
	}

	c.sem <- struct{}{}
	defer func() { <-c.sem }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(baseURL, id), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN item %d: HTTP %s", id, resp.Status)
	}
	var item model.HNItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, fmt.Errorf("HN item %d not found", id)
	}
	if c.Cache != nil {
		_ = c.Cache.WriteJSON("items", id, &item)
	}
	return &item, nil
}
