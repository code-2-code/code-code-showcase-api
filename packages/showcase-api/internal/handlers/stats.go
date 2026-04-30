package handlers

import (
	"net/http"

	providerservicev1 "code-code.internal/go-contract/platform/provider/v1"
	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"code-code.internal/showcase-api/internal/httpjson"
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
		stats.ProviderCount = len(providerResp.GetItems())
		for _, p := range providerResp.GetItems() {
			if p.GetSurfaceId() != "" {
				stats.SurfaceCount++
				if p.GetStatus() != nil && p.GetStatus().GetPhase() == providerservicev1.ProviderPhase_PROVIDER_PHASE_READY {
					stats.ReadyCount++
				}
			}
		}

		httpjson.WriteJSON(w, http.StatusOK, stats)
	}))
}
