package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"code-code.internal/showcase-api/internal/httpjson"
)

// RegisterModelProxyHandlers registers a reverse proxy to the model Connect
// service. The model catalog is fully public metadata; no field projection is
// needed.
//
// Two path prefixes are supported:
//   - /api/models/ — simple REST-style JSON proxy (GET only)
//   - /api/connect/ — Connect protocol proxy (POST, used by connect-web client)
func RegisterModelProxyHandlers(mux *http.ServeMux, modelConnectBaseURL string) {
	baseURL := strings.TrimRight(modelConnectBaseURL, "/")
	if baseURL == "" {
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}

	// REST-style GET proxy (legacy path)
	mux.HandleFunc("/api/models/", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		suffix := strings.TrimPrefix(r.URL.Path, "/api/models")
		proxyModelRequest(w, r, client, baseURL, suffix, http.MethodGet)
	}))

	// Connect protocol proxy — supports POST for binary/JSON connect-web calls.
	// Only ModelService methods (ListModels, etc.) are proxied; no write
	// endpoints exist on this service.
	mux.HandleFunc("/api/connect/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			httpjson.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "only GET and POST are allowed")
			return
		}
		suffix := strings.TrimPrefix(r.URL.Path, "/api/connect")
		proxyConnectRequest(w, r, client, baseURL, suffix)
	})
}

func proxyModelRequest(w http.ResponseWriter, r *http.Request, client *http.Client, baseURL, suffix, method string) {
	target := fmt.Sprintf("%s%s", baseURL, suffix)
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	proxyReq, err := http.NewRequestWithContext(r.Context(), method, target, nil)
	if err != nil {
		httpjson.WriteError(w, http.StatusBadGateway, "proxy_request_failed", err.Error())
		return
	}
	proxyReq.Header.Set("Accept", "application/json")

	resp, err := client.Do(proxyReq)
	if err != nil {
		httpjson.WriteError(w, http.StatusBadGateway, "proxy_upstream_failed", "model service unavailable")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 10<<20))
}

func proxyConnectRequest(w http.ResponseWriter, r *http.Request, client *http.Client, baseURL, suffix string) {
	target := fmt.Sprintf("%s%s", baseURL, suffix)
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	proxyReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpjson.WriteError(w, http.StatusBadGateway, "proxy_request_failed", err.Error())
		return
	}
	// Forward Connect protocol headers.
	for _, header := range []string{"Content-Type", "Connect-Protocol-Version", "Connect-Timeout-Ms"} {
		if value := r.Header.Get(header); value != "" {
			proxyReq.Header.Set(header, value)
		}
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		httpjson.WriteError(w, http.StatusBadGateway, "proxy_upstream_failed", "model service unavailable")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	for _, header := range []string{"Content-Type", "Connect-Status"} {
		if value := resp.Header.Get(header); value != "" {
			w.Header().Set(header, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 10<<20))
}
