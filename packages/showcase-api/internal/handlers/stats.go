package handlers

import (
	"net/http"

	"code-code.internal/showcase-api/internal/httpjson"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
)

// ShowcaseStats carries aggregated platform statistics for the showcase dashboard.
type ShowcaseStats struct {
	VendorCount   int `json:"vendorCount"`
	CLICount      int `json:"cliCount"`
	ProviderCount int `json:"providerCount"`
	SurfaceCount  int `json:"surfaceCount"`
	ReadyCount    int `json:"readyCount"`
}

// RegisterStatsHandlers registers the aggregated stats endpoint.
func RegisterStatsHandlers(mux *http.ServeMux, provider providerservicev1.ProviderServiceClient, support supportv1.SupportServiceClient) {
	mux.HandleFunc("/api/stats", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		stats := ShowcaseStats{}

		vendorResp, err := support.ListVendors(ctx, &supportv1.ListVendorsRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_vendors_failed", err)
			return
		}
		stats.VendorCount = len(vendorResp.GetItems())

		cliResp, err := provider.ListCLIDefinitions(ctx, &providerservicev1.ListCLIDefinitionsRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_clis_failed", err)
			return
		}
		stats.CLICount = len(cliResp.GetItems())

		providerResp, err := provider.ListProviders(ctx, &providerservicev1.ListProvidersRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_providers_failed", err)
			return
		}
		mgmt := &managementv1.ListProvidersResponse{}
		if err := transcodeMessage(providerResp, mgmt); err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "transcode_providers_failed", err)
			return
		}
		stats.ProviderCount = len(mgmt.GetItems())
		for _, p := range mgmt.GetItems() {
			for _, s := range p.GetSurfaces() {
				stats.SurfaceCount++
				if s.GetStatus() != nil && s.GetStatus().GetPhase() == managementv1.ProviderSurfaceBindingPhase_PROVIDER_SURFACE_BINDING_PHASE_READY {
					stats.ReadyCount++
				}
			}
		}

		httpjson.WriteJSON(w, http.StatusOK, stats)
	}))
}
