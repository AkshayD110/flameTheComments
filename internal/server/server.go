package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"hn-flame/internal/hn"
)

//go:embed static/webdist
var embedded embed.FS

type Server struct {
	Addr   string
	Client *hn.Client
	mux    *http.ServeMux
}

func New(addr string, client *hn.Client) *Server {
	s := &Server{Addr: addr, Client: client, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ListenAndServe() error {
	log.Printf("hn-flame listening on http://%s", s.Addr)
	return http.ListenAndServe(s.Addr, s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("GET /api/thread/", s.handleThread)
	s.mux.HandleFunc("/", s.handleStatic)
}

func (s *Server) handleThread(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/thread/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "invalid HN item id", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil || id <= 0 {
		http.Error(w, "invalid HN item id", http.StatusBadRequest)
		return
	}

	if len(parts) == 2 && parts[1] == "initial" {
		thread, err := hn.FetchInitialThread(r.Context(), s.Client, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, thread)
		return
	}

	if len(parts) == 3 && parts[1] == "subtree" {
		commentID, err := strconv.Atoi(parts[2])
		if err != nil || commentID <= 0 {
			http.Error(w, "invalid comment id", http.StatusBadRequest)
			return
		}
		node, err := hn.FetchSubtree(r.Context(), s.Client, commentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, node)
		return
	}

	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	thread, err := hn.FetchThread(r.Context(), s.Client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, thread)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	dist, err := fs.Sub(embedded, "static/webdist")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || strings.HasPrefix(path, "item/") {
		path = "index.html"
	}
	if _, err := fs.Stat(dist, path); err != nil {
		path = "index.html"
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFileFS(w, r, dist, path)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(value)
}

func URL(addr string, id int) string {
	return fmt.Sprintf("http://%s/item/%d", addr, id)
}

func ShutdownOnContext(ctx context.Context, srv *http.Server) {
	<-ctx.Done()
	_ = srv.Shutdown(context.Background())
}
