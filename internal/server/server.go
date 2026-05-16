package server

import (
	"net/http"

	"github.com/yourname/nimbus/internal/router"
	"github.com/yourname/nimbus/internal/storage"
	awsdynamo "github.com/yourname/nimbus/providers/aws/dynamodb"
	awss3 "github.com/yourname/nimbus/providers/aws/s3"
	azureblob "github.com/yourname/nimbus/providers/azure/blob"
	gcpstorage "github.com/yourname/nimbus/providers/gcp/storage"
)

type Config struct {
	Port    string
	DataDir string
}

type Server struct {
	cfg    Config
	router *router.Router
}

// New builds the server and registers every provider.
// Adding a new cloud = implement the Provider interface and register it here.
func New(cfg Config) *Server {
	store := storage.NewMemoryStore() // swap for storage.NewDiskStore(cfg.DataDir) for persistence

	r := router.New()
	// Order matters: more-specific matchers first.
	r.Register(awsdynamo.New(store))
	r.Register(awss3.New(store))
	r.Register(gcpstorage.New(store))
	r.Register(azureblob.New(store))

	return &Server{cfg: cfg, router: r}
}

func (s *Server) Run() error {
	return http.ListenAndServe(":"+s.cfg.Port, s.router)
}
