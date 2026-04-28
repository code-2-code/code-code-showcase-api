package httpjson

import (
	"encoding/json"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const corsAllowedHeaders = "Accept,Content-Type,Authorization,Connect-Protocol-Version,Connect-Timeout-Ms"

// ErrorResponse describes one structured HTTP error payload.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes one JSON response with the provided status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

var protoJSONMarshaler = protojson.MarshalOptions{EmitUnpopulated: true}

// WriteProtoJSON writes one proto message as JSON with camelCase field names.
func WriteProtoJSON(w http.ResponseWriter, status int, msg proto.Message) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if msg == nil {
		return
	}
	data, err := protoJSONMarshaler.Marshal(msg)
	if err != nil {
		return
	}
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n"))
}

// WriteError writes one structured JSON error response.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// WriteServiceError writes one service error with normalized HTTP status.
func WriteServiceError(w http.ResponseWriter, fallbackStatus int, code string, err error) {
	status := statusForServiceError(err, fallbackStatus)
	WriteError(w, status, code, messageForServiceError(err, status))
}

func statusForServiceError(err error, fallbackStatus int) int {
	if err == nil {
		if fallbackStatus >= http.StatusBadRequest {
			return fallbackStatus
		}
		return http.StatusInternalServerError
	}
	if grpcStatus, ok := grpcstatus.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.OK:
			return fallbackStatus
		case codes.InvalidArgument, codes.OutOfRange:
			return http.StatusBadRequest
		case codes.NotFound:
			return http.StatusNotFound
		case codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition:
			return http.StatusConflict
		case codes.PermissionDenied:
			return http.StatusForbidden
		case codes.Unauthenticated:
			return http.StatusUnauthorized
		case codes.ResourceExhausted:
			return http.StatusTooManyRequests
		case codes.Unavailable, codes.DeadlineExceeded:
			return http.StatusServiceUnavailable
		default:
			return http.StatusInternalServerError
		}
	}
	if fallbackStatus >= http.StatusInternalServerError {
		return fallbackStatus
	}
	return http.StatusInternalServerError
}

func messageForServiceError(err error, status int) string {
	if err == nil {
		if text := strings.TrimSpace(http.StatusText(status)); text != "" {
			return strings.ToLower(text)
		}
		return "request failed"
	}
	if status >= http.StatusInternalServerError {
		return "internal server error"
	}
	if grpcStatus, ok := grpcstatus.FromError(err); ok {
		if message := strings.TrimSpace(grpcStatus.Message()); message != "" {
			return message
		}
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		if text := strings.TrimSpace(http.StatusText(status)); text != "" {
			return strings.ToLower(text)
		}
		return "request failed"
	}
	return message
}

// WithCORS applies a permissive development CORS policy for the showcase API.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireGET is a middleware that rejects non-GET requests.
func RequireGET(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		handler(w, r)
	}
}
