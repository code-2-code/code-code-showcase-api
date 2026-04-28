package handlers

import (
	"net/http"

	"code-code.internal/showcase-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)

// RegisterProviderHandlers registers projected, read-only provider endpoints.
// The response shape is ListProvidersResponse (protobuf JSON) to match
// the existing console-web SWR hooks, but sensitive fields are stripped.
func RegisterProviderHandlers(mux *http.ServeMux, provider providerservicev1.ProviderServiceClient) {
	mux.HandleFunc("/api/providers", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		response, err := provider.ListProviders(r.Context(), &providerservicev1.ListProvidersRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_providers_failed", err)
			return
		}
		mgmt := &managementv1.ListProvidersResponse{}
		if err := transcodeMessage(response, mgmt); err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "transcode_providers_failed", err)
			return
		}
		stripSensitiveFields(mgmt)
		httpjson.WriteProtoJSON(w, http.StatusOK, mgmt)
	}))
}

// stripSensitiveFields removes account-identifying and credential fields from
// each ProviderView while retaining display_name, vendor, status, and model
// catalog. The message is modified in place.
func stripSensitiveFields(response *managementv1.ListProvidersResponse) {
	for _, item := range response.GetItems() {
		// Strip account instance identifiers.
		item.ProviderId = ""
		item.ProviderCredentialId = ""
		item.CredentialSubjectSummary = nil

		for _, surface := range item.GetSurfaces() {
			// Strip credential reference and custom endpoint URLs.
			surface.ProviderCredentialId = ""
			surface.ProviderId = ""
			if surface.GetRuntime() != nil && surface.GetRuntime().GetApi() != nil {
				surface.GetRuntime().GetApi().BaseUrl = ""
			}
		}
	}
}
