package router

import (
	"fmt"
	"net/http"
	"strings"
)

// Provider is the contract every cloud service emulator implements.
//
// Matches() inspects the request and returns true if THIS provider should
// handle it. ServeHTTP() then handles the actual API call.
//
// The router walks providers in registration order. First match wins.
type Provider interface {
	Name() string                 // e.g. "aws-s3", "gcp-storage"
	Matches(r *http.Request) bool // does this request belong to me?
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type Router struct {
	providers []Provider
}

func New() *Router {
	return &Router{}
}

func (rt *Router) Register(p Provider) {
	rt.providers = append(rt.providers, p)
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Health endpoint so users can confirm Nimbus is up.
	if r.URL.Path == "/_nimbus/health" {
		fmt.Fprintln(w, "ok")
		return
	}

	if r.URL.Path == "/_nimbus/providers" {
		for _, p := range rt.providers {
			fmt.Fprintln(w, p.Name())
		}
		return
	}

	for _, p := range rt.providers {
		if p.Matches(r) {
			p.ServeHTTP(w, r)
			return
		}
	}

	http.Error(w, fmt.Sprintf("no provider matched: %s %s (host=%s)",
		r.Method, r.URL.Path, strings.Split(r.Host, ":")[0]), http.StatusNotFound)
}
