package handlers

import (
	"net/http"

	"code-code.internal/showcase-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
)

// RegisterCLIHandlers registers read-only CLI definition endpoints.
func RegisterCLIHandlers(mux *http.ServeMux, provider providerservicev1.ProviderServiceClient) {
	mux.HandleFunc("/api/clis", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		response, err := provider.ListCLIDefinitions(r.Context(), &providerservicev1.ListCLIDefinitionsRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_clis_failed", err)
			return
		}
		// CLIDefinitionView contains only public metadata: cli_id,
		// display_name, icon_url, website_url, description, capabilities.
		out := &managementv1.ListCLIDefinitionsResponse{}
		if err := transcodeMessage(response, out); err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "transcode_clis_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, out)
	}))
}
