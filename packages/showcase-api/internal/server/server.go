package server

import (
	"fmt"
	"net/http"

	"code-code.internal/showcase-api/internal/handlers"
	"code-code.internal/showcase-api/internal/httpjson"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"google.golang.org/grpc"
)

// Config groups dependencies required to assemble the showcase API.
type Config struct {
	ProviderConn        grpc.ClientConnInterface
	SupportConn         grpc.ClientConnInterface
	ModelConnectBaseURL string
	PrometheusBaseURL   string
}

// Server bundles the HTTP handler for the read-only showcase API.
type Server struct {
	Handler http.Handler
}

// New creates one showcase API server. All registered endpoints are read-only.
func New(config Config) (*Server, error) {
	if config.ProviderConn == nil {
		return nil, fmt.Errorf("showcase/server: provider connection is nil")
	}
	if config.SupportConn == nil {
		return nil, fmt.Errorf("showcase/server: support connection is nil")
	}

	providerClient := providerservicev1.NewProviderServiceClient(config.ProviderConn)
	supportClient := supportv1.NewSupportServiceClient(config.SupportConn)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/healthz", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}))
	mux.HandleFunc("/api/readyz", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}))

	handlers.RegisterVendorHandlers(mux, supportClient)
	handlers.RegisterCLIHandlers(mux, providerClient)
	handlers.RegisterProviderHandlers(mux, providerClient)
	handlers.RegisterStatsHandlers(mux, providerClient, supportClient)
	handlers.RegisterModelProxyHandlers(mux, config.ModelConnectBaseURL)

	return &Server{Handler: httpjson.WithCORS(mux)}, nil
}
