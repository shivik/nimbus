# Nimbus ☁️

A small, hackable, **multi-cloud local emulator** written in Go.
Run one binary (or one container), get fake AWS + GCP + Azure endpoints on
`localhost:4566`. Point your SDK at it. Develop without touching real cloud.

> Status: **starter / foundation**. Implements a working slice of S3,
> DynamoDB, GCS, and Azure Blob. The architecture is designed so adding
> a new cloud or service is one new file.

---

## Why another emulator?

| | LocalStack | Floci | **Nimbus** |
|---|---|---|---|
| Clouds supported | AWS | AWS | **AWS + GCP + Azure (pluggable)** |
| Language | Python | Java | **Go (single binary)** |
| Lines of code | ~500k | ~100k | **~700** |
| Easy to read & extend | ❌ | ⚠️ | **✅** |
| Production-ready coverage | ✅ | ✅ | ❌ (yet) |

Nimbus is intentionally *small*. The point is for you to **understand
every line** and add what you need.

---

## Quick start

```bash
# Option A: run with Go
go run ./cmd/nimbus

# Option B: docker
docker compose up
```

Then in another terminal:

```bash
chmod +x smoke_test.sh && ./smoke_test.sh
```

You should see PUT/GET/DELETE round-tripping against all four providers.

---

## Architecture (one screen)

```
┌────────────────┐   HTTP    ┌──────────────────────────────┐
│ Your SDK / CLI │ ────────► │           Nimbus :4566        │
└────────────────┘           │                               │
                             │   ┌─────────────────────┐    │
                             │   │ Router              │    │
                             │   │ (first match wins)  │    │
                             │   └──────────┬──────────┘    │
                             │              │               │
                             │   ┌──────────▼──────────┐    │
                             │   │ Providers           │    │
                             │   │  • aws/s3           │    │
                             │   │  • aws/dynamodb     │    │
                             │   │  • gcp/storage      │    │
                             │   │  • azure/blob       │    │
                             │   └──────────┬──────────┘    │
                             │              │               │
                             │   ┌──────────▼──────────┐    │
                             │   │ Storage (memory)    │    │
                             │   └─────────────────────┘    │
                             └──────────────────────────────┘
```

**Three concepts, that's it:**

1. **Router** — looks at each request, asks each provider "is this yours?"
2. **Provider** — anything that implements `Name() / Matches() / ServeHTTP()`.
3. **Store** — a tiny key/value interface every provider writes to.

---

## Project layout

```
nimbus/
├── cmd/nimbus/main.go          # entry point
├── internal/
│   ├── server/server.go        # wires router + providers + store
│   ├── router/router.go        # dispatches to the right provider
│   └── storage/storage.go      # shared K/V abstraction (memory impl)
└── providers/
    ├── aws/s3/                 # AWS S3
    ├── aws/dynamodb/           # AWS DynamoDB
    ├── gcp/storage/            # Google Cloud Storage
    └── azure/blob/             # Azure Blob Storage
```

---

## Adding a new cloud or service

1. Make a folder under `providers/<cloud>/<service>/`
2. Implement three methods:

```go
type Provider interface {
    Name() string                       // for /_nimbus/providers
    Matches(r *http.Request) bool       // does this request belong to me?
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

3. Register it in `internal/server/server.go`:

```go
r.Register(myprovider.New(store))
```

That's the whole extension model. `Matches()` typically checks a unique
header or path prefix — AWS uses `Authorization: AWS4-...` and
`X-Amz-Target`, GCP uses `/storage/v1/...` paths, Azure uses
`x-ms-version`, etc.

---

## What's implemented

| Cloud | Service | Operations |
|---|---|---|
| AWS | S3 | create bucket, put/get/delete/list objects |
| AWS | DynamoDB | CreateTable, PutItem, GetItem, DeleteItem, Scan |
| GCP | Cloud Storage | upload, download, delete, list |
| Azure | Blob Storage | create container, put/get/delete/list blobs |

---

## What's NOT implemented (yet)

A lot. This is honest:

- No SigV4 signature *verification* (the AWS providers accept anything)
- No persistence (memory store only — restart loses data)
- No IAM, no auth, no quotas
- No streams, queues, lambda, pub/sub
- Limited error codes (real clouds have very specific error XML/JSON)

These are all incremental additions following the same pattern.

---

## Roadmap ideas (good first PRs)

- [ ] Disk-backed `Store` so data survives restarts
- [ ] AWS SQS (in-memory queues)
- [ ] GCP Pub/Sub
- [ ] Azure Cosmos DB
- [ ] Web UI showing live state (`/_nimbus/ui`)
- [ ] Cloudflare R2 (S3-compatible — should be trivial)
- [ ] DigitalOcean Spaces (S3-compatible)

---

## License

MIT.
