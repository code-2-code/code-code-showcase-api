package handlers

import (
	"net/http"

	"code-code.internal/showcase-api/internal/httpjson"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

// RegisterVendorHandlers registers read-only vendor listing endpoints.
// Both /api/vendors and /api/support/vendors are registered so the
// existing console-web SWR hooks work without changes.
func RegisterVendorHandlers(mux *http.ServeMux, support supportv1.SupportServiceClient) {
	handler := httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		response, err := support.ListVendors(r.Context(), &supportv1.ListVendorsRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_vendors_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("/api/vendors", handler)
	mux.HandleFunc("/api/support/vendors", handler)

	clisHandler := httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		response, err := support.ListCLIs(r.Context(), &supportv1.ListCLIsRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_support_clis_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("/api/support/clis", clisHandler)

	surfacesHandler := httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		response, err := support.ListProviderSurfaces(r.Context(), &supportv1.ListProviderSurfacesRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_provider_surfaces_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("/api/support/provider-surfaces", surfacesHandler)
}
