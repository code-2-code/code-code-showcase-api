package handlers

import (
	"net/http"

	supportv1 "code-code.internal/go-contract/platform/support/v1"
	"code-code.internal/showcase-api/internal/httpjson"
)

// RegisterCLIHandlers registers read-only CLI endpoints.
func RegisterCLIHandlers(mux *http.ServeMux, support supportv1.SupportServiceClient) {
	mux.HandleFunc("/api/clis", httpjson.RequireGET(func(w http.ResponseWriter, r *http.Request) {
		response, err := support.ListCLIs(r.Context(), &supportv1.ListCLIsRequest{})
		if err != nil {
			httpjson.WriteServiceError(w, http.StatusInternalServerError, "list_clis_failed", err)
			return
		}
		httpjson.WriteProtoJSON(w, http.StatusOK, response)
	}))
}
