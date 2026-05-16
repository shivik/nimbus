// Package storage emulates a tiny subset of Google Cloud Storage (GCS).
// GCS uses a REST JSON API at /storage/v1/b/<bucket>/o/<object>.
package storage

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	nstore "github.com/yourname/nimbus/internal/storage"
)

const namespace = "gcp-storage"

type Provider struct {
	store nstore.Store
}

func New(store nstore.Store) *Provider {
	return &Provider{store: store}
}

func (p *Provider) Name() string { return "gcp-storage" }

// GCS requests start with /storage/v1/ OR /upload/storage/v1/.
func (p *Provider) Matches(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/storage/v1/") ||
		strings.HasPrefix(r.URL.Path, "/upload/storage/v1/")
}

type object struct {
	Bucket string `json:"bucket"`
	Name   string `json:"name"`
	Size   int    `json:"size"`
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Upload path: POST /upload/storage/v1/b/<bucket>/o?name=<key>
	if strings.HasPrefix(r.URL.Path, "/upload/storage/v1/b/") && r.Method == http.MethodPost {
		bucket := extractBucket(r.URL.Path, "/upload/storage/v1/b/")
		name := r.URL.Query().Get("name")
		body, _ := io.ReadAll(r.Body)
		_ = p.store.Put(namespace+":"+bucket, name, body)
		_ = json.NewEncoder(w).Encode(object{Bucket: bucket, Name: name, Size: len(body)})
		return
	}

	// Standard path: /storage/v1/b/<bucket>/o/<object>
	if !strings.HasPrefix(r.URL.Path, "/storage/v1/b/") {
		http.NotFound(w, r)
		return
	}
	tail := strings.TrimPrefix(r.URL.Path, "/storage/v1/b/")
	parts := strings.SplitN(tail, "/o", 2)
	bucket := parts[0]
	objName := ""
	if len(parts) > 1 {
		objName = strings.TrimPrefix(parts[1], "/")
	}

	switch r.Method {
	case http.MethodGet:
		if objName == "" {
			// list objects in bucket
			keys, _ := p.store.List(namespace+":"+bucket, "")
			items := []object{}
			for _, k := range keys {
				items = append(items, object{Bucket: bucket, Name: k})
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
			return
		}
		// Download: ?alt=media returns raw bytes
		val, ok, _ := p.store.Get(namespace+":"+bucket, objName)
		if !ok {
			http.Error(w, `{"error":{"code":404,"message":"Not Found"}}`, http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("alt") == "media" {
			_, _ = w.Write(val)
			return
		}
		_ = json.NewEncoder(w).Encode(object{Bucket: bucket, Name: objName, Size: len(val)})

	case http.MethodDelete:
		_ = p.store.Delete(namespace+":"+bucket, objName)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func extractBucket(path, prefix string) string {
	tail := strings.TrimPrefix(path, prefix)
	return strings.SplitN(tail, "/", 2)[0]
}
