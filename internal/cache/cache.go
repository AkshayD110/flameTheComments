package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileCache struct {
	Root string
	TTL  time.Duration
}

func New(root string, ttl time.Duration) *FileCache {
	return &FileCache{Root: root, TTL: ttl}
}

func (c *FileCache) ReadJSON(kind string, id int, out any, refresh bool) bool {
	if refresh {
		return false
	}
	path := c.path(kind, id)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || time.Since(info.ModTime()) > c.TTL {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return json.Unmarshal(b, out) == nil
}

func (c *FileCache) WriteJSON(kind string, id int, value any) error {
	path := c.path(kind, id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (c *FileCache) path(kind string, id int) string {
	return filepath.Join(c.Root, kind, fmt.Sprintf("%d.json", id))
}

func DefaultRoot() string {
	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		return filepath.Join(dir, "hn-flame")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".cache", "hn-flame")
	}
	return ".cache/hn-flame"
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}
