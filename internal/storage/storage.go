package storage

import "sync"

// Store is the single key/value abstraction every provider uses.
// Real disk-backed store, Redis-backed store, etc. can implement this
// same interface without changing any provider code.
type Store interface {
	Put(namespace, key string, value []byte) error
	Get(namespace, key string) ([]byte, bool, error)
	Delete(namespace, key string) error
	List(namespace, prefix string) ([]string, error)
}

// MemoryStore is the simplest implementation - everything in RAM.
// Good for tests; data is lost on restart.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]map[string][]byte // namespace -> key -> value
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]map[string][]byte)}
}

func (m *MemoryStore) Put(ns, key string, val []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.data[ns]; !ok {
		m.data[ns] = make(map[string][]byte)
	}
	m.data[ns][key] = val
	return nil
}

func (m *MemoryStore) Get(ns, key string) ([]byte, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if bucket, ok := m.data[ns]; ok {
		if v, ok := bucket[key]; ok {
			return v, true, nil
		}
	}
	return nil, false, nil
}

func (m *MemoryStore) Delete(ns, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if bucket, ok := m.data[ns]; ok {
		delete(bucket, key)
	}
	return nil
}

func (m *MemoryStore) List(ns, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []string
	if bucket, ok := m.data[ns]; ok {
		for k := range bucket {
			if prefix == "" || hasPrefix(k, prefix) {
				out = append(out, k)
			}
		}
	}
	return out, nil
}

func hasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
