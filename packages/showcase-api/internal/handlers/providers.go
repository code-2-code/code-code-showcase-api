package handlers

import (
	"net/http"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	"code-code.internal/showcase-api/internal/httpjson"
)

// RegisterProviderHandlers registers projected, read-only provider endpoints
// with account and credential identifiers stripped for the public showcase.
func RegisterProviderHandlers(
	mux *http.ServeMux,
	provider providerservicev1.ProviderServiceClient,
	hostTelemetry *ProviderHostTelemetryClient,
) {
	mux.HandleFunc("/api/providers/surfaces", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		response, err := provider.ListProviderSurfaces(r.Context(), &providerservicev1.ListProviderSurfacesRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_provider_surfaces_failed", err)
			return
		}
		out := &managementv1.ListProviderSurfacesResponse{}
		if err := transcodeMessage(response, out); err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "project_provider_surfaces_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, out)
	}))

	mux.HandleFunc("/api/providers", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		response, err := provider.ListProviders(r.Context(), &providerservicev1.ListProvidersRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_providers_failed", err)
			return
		}
		out := &managementv1.ListProvidersResponse{}
		if err := transcodeMessage(response, out); err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "project_providers_failed", err)
			return
		}
		attachProviderHostTelemetry(r.Context(), out.GetItems(), hostTelemetry)
		stripSensitiveFields(out)
		httpjson.WriteProtoJSON(w, http.StatusOK, out)
	}))
}

// stripSensitiveFields removes account-identifying and credential fields from
// each ProviderView while retaining display_name, status, and model catalog.
// The message is modified in place.
func stripSensitiveFields(response *managementv1.ListProvidersResponse) {
	for _, item := range response.GetItems() {
		// Strip account instance identifiers.
		item.ProviderId = ""
		item.ProviderCredentialId = ""

		// Strip custom endpoint URLs.
		for _, endpoint := range item.GetEndpoints() {
			if endpoint.GetApi() != nil {
				endpoint.GetApi().BaseUrl = ""
			}
		}
	}
}
