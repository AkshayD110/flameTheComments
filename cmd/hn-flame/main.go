package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"hn-flame/internal/browser"
	"hn-flame/internal/cache"
	"hn-flame/internal/hn"
	"hn-flame/internal/server"
)

func main() {
	var port int
	var refresh bool
	var noOpen bool
	var cacheDir string
	flag.IntVar(&port, "port", 3000, "localhost port to serve the UI")
	flag.BoolVar(&refresh, "refresh", false, "ignore cached HN items and fetch fresh data")
	flag.BoolVar(&noOpen, "no-open", false, "do not open the browser automatically")
	flag.StringVar(&cacheDir, "cache-dir", cache.DefaultRoot(), "cache directory")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: hn-flame [--port 3000] [--refresh] <hn-item-id-or-url>")
		os.Exit(2)
	}
	id, err := parseItemID(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	if !portAvailable(addr) {
		log.Fatalf("port is already in use: %s", addr)
	}

	fc := cache.New(cacheDir, time.Hour)
	client := hn.NewClient(fc, refresh, 12)
	s := server.New(addr, client)
	uiURL := server.URL(addr, id)
	if !noOpen {
		go func() {
			time.Sleep(300 * time.Millisecond)
			if err := browser.Open(uiURL); err != nil {
				log.Printf("open browser: %v", err)
			}
		}()
	}
	fmt.Printf("Opening %s\n", uiURL)
	fmt.Printf("Cache: %s\n", cacheDir)
	log.Fatal(s.ListenAndServe())
}

func parseItemID(input string) (int, error) {
	input = strings.TrimSpace(input)
	if id, err := strconv.Atoi(input); err == nil && id > 0 {
		return id, nil
	}
	u, err := url.Parse(input)
	if err != nil || u.Host == "" {
		return 0, fmt.Errorf("invalid item id or URL: %q", input)
	}
	if idStr := u.Query().Get("id"); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err == nil && id > 0 {
			return id, nil
		}
	}
	return 0, fmt.Errorf("could not find positive item id in %q", input)
}

func portAvailable(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
