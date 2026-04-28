package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code-code.internal/showcase-api/internal/server"
	"code-code.internal/showcase-api/internal/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := envOrDefault("SHOWCASE_API_ADDR", ":8080")
	providerAddr := envOrDefault("SHOWCASE_PROVIDER_GRPC_ADDR", "platform-provider-service:8081")
	supportAddr := envOrDefault("SHOWCASE_SUPPORT_GRPC_ADDR", "platform-support-service:8081")
	modelConnectBaseURL := envOrDefault("SHOWCASE_MODEL_CONNECT_BASE_URL", "http://platform-model-service:8080")
	prometheusBaseURL := envOrDefault("SHOWCASE_PROMETHEUS_BASE_URL", "http://prometheus.code-code-observability.svc.cluster.local:9090")

	telemetryShutdown, err := telemetry.Setup(context.Background(), envOrDefault("OTEL_SERVICE_NAME", "showcase-api"))
	must(err)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryShutdown(shutdownCtx); err != nil {
			log.Printf("shutdown telemetry failed: %v", err)
		}
	}()

	providerConn, err := grpc.NewClient(
		providerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := providerConn.Close(); err != nil {
			log.Printf("close provider connection failed: %v", err)
		}
	}()

	supportConn, err := grpc.NewClient(
		supportAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := supportConn.Close(); err != nil {
			log.Printf("close support connection failed: %v", err)
		}
	}()

	srv, err := server.New(server.Config{
		ProviderConn:        providerConn,
		SupportConn:         supportConn,
		ModelConnectBaseURL: modelConnectBaseURL,
		PrometheusBaseURL:   prometheusBaseURL,
	})
	must(err)

	apiServer := &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(srv.Handler, "showcase_api_http"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down showcase-api...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("showcase-api shutdown failed: %v", err)
		}
	}()

	log.Printf("showcase-api listening on %s (provider=%s, support=%s, model_connect=%s, prometheus=%s)",
		addr, providerAddr, supportAddr, modelConnectBaseURL, prometheusBaseURL)
	if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		must(err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
