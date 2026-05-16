// Package s3 emulates a tiny subset of AWS S3.
// Supports: create bucket, put object, get object, delete object, list objects.
package s3

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/yourname/nimbus/internal/storage"
)

const namespace = "aws-s3"

type Provider struct {
	store storage.Store
}

func New(store storage.Store) *Provider {
	return &Provider{store: store}
}

func (p *Provider) Name() string { return "aws-s3" }

// Matches AWS S3 requests by Authorization header (SigV4 for "s3") OR a path
// shaped like /bucket/key. We keep it loose so test clients without proper
// SigV4 still work.
func (p *Provider) Matches(r *http.Request) bool {
	// IMPORTANT: don't claim DynamoDB / other JSON-API AWS services.
	// Those carry X-Amz-Target and we let their providers handle them.
	if r.Header.Get("X-Amz-Target") != "" {
		return false
	}

	auth := r.Header.Get("Authorization")
	if strings.Contains(auth, "AWS4") && strings.Contains(auth, "/s3/") {
		return true
	}
	// Virtual-host style: bucket.s3.amazonaws.com
	if strings.Contains(r.Host, ".s3.") {
		return true
	}
	// Any x-amz-* header (excluding the target header handled above) signals S3.
	for h := range r.Header {
		lh := strings.ToLower(h)
		if strings.HasPrefix(lh, "x-amz-") && lh != "x-amz-target" {
			return true
		}
	}
	return false
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Path-style: /bucket/key...   Virtual-host style: bucket.s3.amazonaws.com/key
	bucket, key := parseBucketKey(r)

	switch r.Method {
	case http.MethodPut:
		if key == "" {
			// Create bucket
			_ = p.store.Put(namespace+":buckets", bucket, []byte{})
			w.WriteHeader(http.StatusOK)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = p.store.Put(namespace+":"+bucket, key, body)
		w.Header().Set("ETag", fmt.Sprintf("\"%x\"", len(body)))
		w.WriteHeader(http.StatusOK)

	case http.MethodGet:
		if key == "" {
			p.listObjects(w, bucket)
			return
		}
		val, ok, _ := p.store.Get(namespace+":"+bucket, key)
		if !ok {
			http.Error(w, "NoSuchKey", http.StatusNotFound)
			return
		}
		_, _ = w.Write(val)

	case http.MethodDelete:
		_ = p.store.Delete(namespace+":"+bucket, key)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type listBucketResult struct {
	XMLName  xml.Name `xml:"ListBucketResult"`
	Name     string   `xml:"Name"`
	Contents []struct {
		Key string `xml:"Key"`
	} `xml:"Contents"`
}

func (p *Provider) listObjects(w http.ResponseWriter, bucket string) {
	keys, _ := p.store.List(namespace+":"+bucket, "")
	result := listBucketResult{Name: bucket}
	for _, k := range keys {
		result.Contents = append(result.Contents, struct {
			Key string `xml:"Key"`
		}{Key: k})
	}
	w.Header().Set("Content-Type", "application/xml")
	_ = xml.NewEncoder(w).Encode(result)
}

func parseBucketKey(r *http.Request) (bucket, key string) {
	// Virtual-host style: bucket.s3.amazonaws.com
	host := strings.Split(r.Host, ":")[0]
	if strings.Contains(host, ".s3.") {
		bucket = strings.Split(host, ".s3.")[0]
		key = strings.TrimPrefix(r.URL.Path, "/")
		return
	}
	// Path style: /bucket/key
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	bucket = parts[0]
	if len(parts) > 1 {
		key = parts[1]
	}
	return
}
