// Package dynamodb emulates a tiny subset of AWS DynamoDB.
// Supports: PutItem, GetItem, DeleteItem, Scan. CreateTable is a no-op
// (tables are created implicitly on first write).
package dynamodb

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/yourname/nimbus/internal/storage"
)

const namespace = "aws-dynamodb"

type Provider struct {
	store storage.Store
}

func New(store storage.Store) *Provider {
	return &Provider{store: store}
}

func (p *Provider) Name() string { return "aws-dynamodb" }

// DynamoDB requests carry an X-Amz-Target header like "DynamoDB_20120810.PutItem".
func (p *Provider) Matches(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("X-Amz-Target"), "DynamoDB_")
}

type request struct {
	TableName string                 `json:"TableName"`
	Item      map[string]interface{} `json:"Item"`
	Key       map[string]interface{} `json:"Key"`
}

func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := r.Header.Get("X-Amz-Target")
	op := target[strings.LastIndex(target, ".")+1:]

	body, _ := io.ReadAll(r.Body)
	var req request
	_ = json.Unmarshal(body, &req)

	w.Header().Set("Content-Type", "application/x-amz-json-1.0")

	switch op {
	case "CreateTable", "DeleteTable":
		_, _ = w.Write([]byte(`{"TableDescription":{"TableStatus":"ACTIVE"}}`))

	case "PutItem":
		keyJSON, _ := json.Marshal(req.Item)
		// Use the JSON string of the whole item as the key. Real DynamoDB
		// uses a hash key; for the emulator this is good enough to round-trip.
		_ = p.store.Put(namespace+":"+req.TableName, string(keyJSON), keyJSON)
		_, _ = w.Write([]byte(`{}`))

	case "GetItem":
		keyJSON, _ := json.Marshal(req.Key)
		// Try direct hit first
		val, ok, _ := p.store.Get(namespace+":"+req.TableName, string(keyJSON))
		if ok {
			_, _ = w.Write([]byte(`{"Item":` + string(val) + `}`))
			return
		}
		// Fallback: scan and match by key fields
		keys, _ := p.store.List(namespace+":"+req.TableName, "")
		for _, k := range keys {
			v, _, _ := p.store.Get(namespace+":"+req.TableName, k)
			var item map[string]interface{}
			_ = json.Unmarshal(v, &item)
			if matchesKey(item, req.Key) {
				_, _ = w.Write([]byte(`{"Item":` + string(v) + `}`))
				return
			}
		}
		_, _ = w.Write([]byte(`{}`))

	case "DeleteItem":
		keyJSON, _ := json.Marshal(req.Key)
		_ = p.store.Delete(namespace+":"+req.TableName, string(keyJSON))
		_, _ = w.Write([]byte(`{}`))

	case "Scan":
		keys, _ := p.store.List(namespace+":"+req.TableName, "")
		items := []json.RawMessage{}
		for _, k := range keys {
			v, _, _ := p.store.Get(namespace+":"+req.TableName, k)
			items = append(items, v)
		}
		resp := map[string]interface{}{
			"Items": items,
			"Count": len(items),
		}
		_ = json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, "unsupported op: "+op, http.StatusBadRequest)
	}
}

func matchesKey(item, key map[string]interface{}) bool {
	for k, v := range key {
		if iv, ok := item[k]; !ok {
			return false
		} else {
			a, _ := json.Marshal(iv)
			b, _ := json.Marshal(v)
			if string(a) != string(b) {
				return false
			}
		}
	}
	return true
}
