// Package blob emulates a tiny subset of Azure Blob Storage.
// Azure uses container/blob path style: /<container>/<blob>
// and signals itself via the "x-ms-version" header.
package blob

import (
	"encoding/xml"
	"io"
	"net/http"
	"strings"

	"github.com/yourname/nimbus/internal/storage"
)

const namespace = "azure-blob"

type Provider struct {
	store storage.Store
}

func New(store storage.Store) *Provider {
	return &Provider{store: store}
}

func (p *Provider) Name() string { return "azure-blob" }

// Azure Storage requests always include the x-ms-version header.
func (p *Provider) Matches(r *http.Request) bool {
	return r.Header.Get("x-ms-version") != ""
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	container := parts[0]
	blob := ""
	if len(parts) > 1 {
		blob = parts[1]
	}

	// Container create is a PUT with ?restype=container and no blob name
	switch r.Method {
	case http.MethodPut:
		if blob == "" {
			_ = p.store.Put(namespace+":containers", container, []byte{})
			w.WriteHeader(http.StatusCreated)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = p.store.Put(namespace+":"+container, blob, body)
		w.WriteHeader(http.StatusCreated)

	case http.MethodGet:
		if blob == "" {
			p.listBlobs(w, container)
			return
		}
		val, ok, _ := p.store.Get(namespace+":"+container, blob)
		if !ok {
			http.Error(w, "BlobNotFound", http.StatusNotFound)
			return
		}
		_, _ = w.Write(val)

	case http.MethodDelete:
		_ = p.store.Delete(namespace+":"+container, blob)
		w.WriteHeader(http.StatusAccepted)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type enumerationResults struct {
	XMLName xml.Name `xml:"EnumerationResults"`
	Blobs   struct {
		Blob []struct {
			Name string `xml:"Name"`
		} `xml:"Blob"`
	} `xml:"Blobs"`
}

func (p *Provider) listBlobs(w http.ResponseWriter, container string) {
	keys, _ := p.store.List(namespace+":"+container, "")
	var result enumerationResults
	for _, k := range keys {
		result.Blobs.Blob = append(result.Blobs.Blob, struct {
			Name string `xml:"Name"`
		}{Name: k})
	}
	w.Header().Set("Content-Type", "application/xml")
	_ = xml.NewEncoder(w).Encode(result)
}
