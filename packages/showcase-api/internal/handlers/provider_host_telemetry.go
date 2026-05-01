package handlers

import (
	"context"
	"encoding/json"
	"math"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const providerHostTelemetryQuery = `last_over_time({job="provider-host-latency",__name__=~"probe_success|probe_duration_seconds|probe_http_status_code"}[5m])`

type ProviderHostTelemetryClient struct {
	baseURL    string
	httpClient *http.Client
}

type providerHostTelemetryTarget struct {
	key       string
	targetURL string
	scheme    string
	host      string
	port      string
}

type providerHostTelemetryPoint struct {
	target     providerHostTelemetryTarget
	success    *float64
	latency    *float64
	statusCode *float64
	sampledAt  *timestamppb.Timestamp
}

type prometheusVectorResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []prometheusVectorSample `json:"result"`
	} `json:"data"`
}

type prometheusVectorSample struct {
	Metric    map[string]string `json:"metric"`
	ValuePair []json.RawMessage `json:"value"`
}

func NewProviderHostTelemetryClient(baseURL string) *ProviderHostTelemetryClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil
	}
	return &ProviderHostTelemetryClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func attachProviderHostTelemetry(ctx context.Context, providers []*managementv1.ProviderView, client *ProviderHostTelemetryClient) {
	targets := providerHostTelemetryTargetsFromProviders(providers)
	if len(targets) == 0 {
		return
	}
	points := map[string]*providerHostTelemetryPoint{}
	if client != nil {
		points = providerHostTelemetryPointsFromSamples(client.queryVector(ctx))
	}
	_ = points
}

func (c *ProviderHostTelemetryClient) queryVector(ctx context.Context) []prometheusVectorSample {
	if c == nil || c.httpClient == nil || c.baseURL == "" {
		return nil
	}
	requestURL := c.baseURL + "/api/v1/query?query=" + url.QueryEscape(providerHostTelemetryQuery)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil
	}
	var payload prometheusVectorResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil
	}
	if payload.Status != "success" {
		return nil
	}
	return payload.Data.Result
}

func providerHostTelemetryTargetsFromProviders(providers []*managementv1.ProviderView) map[string]providerHostTelemetryTarget {
	targets := map[string]providerHostTelemetryTarget{}
	for _, provider := range providers {
		target, ok := providerHostTelemetryTargetFromProvider(provider)
		if ok {
			targets[target.key] = target
		}
	}
	return targets
}

func providerHostTelemetryPointsFromSamples(samples []prometheusVectorSample) map[string]*providerHostTelemetryPoint {
	points := map[string]*providerHostTelemetryPoint{}
	for _, sample := range samples {
		target, ok := providerHostTelemetryTargetFromMetric(sample.Metric)
		if !ok {
			continue
		}
		point := points[target.key]
		if point == nil {
			point = &providerHostTelemetryPoint{target: target}
			points[target.key] = point
		}
		value, timestamp, ok := sample.value()
		if !ok {
			continue
		}
		if point.sampledAt == nil || timestamp.After(point.sampledAt.AsTime()) {
			point.sampledAt = timestamppb.New(timestamp)
		}
		switch strings.TrimSpace(sample.Metric["__name__"]) {
		case "probe_success":
			point.success = &value
		case "probe_duration_seconds":
			point.latency = &value
		case "probe_http_status_code":
			point.statusCode = &value
		}
	}
	return points
}

func (s prometheusVectorSample) value() (float64, time.Time, bool) {
	if len(s.ValuePair) != 2 {
		return 0, time.Time{}, false
	}
	var ts float64
	if err := json.Unmarshal(s.ValuePair[0], &ts); err != nil {
		return 0, time.Time{}, false
	}
	var raw string
	if err := json.Unmarshal(s.ValuePair[1], &raw); err != nil {
		return 0, time.Time{}, false
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, time.Time{}, false
	}
	sec, frac := math.Modf(ts)
	return value, time.Unix(int64(sec), int64(frac*1e9)), true
}

func providerHostTelemetryTargetFromMetric(metric map[string]string) (providerHostTelemetryTarget, bool) {
	if len(metric) == 0 {
		return providerHostTelemetryTarget{}, false
	}
	scheme := strings.ToLower(strings.TrimSpace(metric["scheme"]))
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(metric["host"])), ".")
	port := strings.TrimSpace(metric["port"])
	if scheme == "" || host == "" || port == "" {
		return normalizeProviderHostTelemetryTarget(metric["instance"])
	}
	return providerHostTelemetryTargetFromParts(scheme, host, port)
}

func providerHostTelemetryTargetFromProvider(provider *managementv1.ProviderView) (providerHostTelemetryTarget, bool) {
	if provider == nil {
		return providerHostTelemetryTarget{}, false
	}
	for _, endpoint := range provider.GetEndpoints() {
		if endpoint.GetApi() != nil {
			return normalizeProviderHostTelemetryTarget(endpoint.GetApi().GetBaseUrl())
		}
	}
	return providerHostTelemetryTarget{}, false
}

func normalizeProviderHostTelemetryTarget(raw string) (providerHostTelemetryTarget, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return providerHostTelemetryTarget{}, false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return providerHostTelemetryTarget{}, false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	host := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(parsed.Hostname())), ".")
	port := strings.TrimSpace(parsed.Port())
	if port == "" {
		port = providerHostTelemetryDefaultPort(scheme)
	}
	return providerHostTelemetryTargetFromParts(scheme, host, port)
}

func providerHostTelemetryTargetFromParts(scheme string, host string, port string) (providerHostTelemetryTarget, bool) {
	if scheme != "http" && scheme != "https" {
		return providerHostTelemetryTarget{}, false
	}
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	port = strings.TrimSpace(port)
	if host == "" || port == "" {
		return providerHostTelemetryTarget{}, false
	}
	targetURL := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, port),
		Path:   "/",
	}
	return providerHostTelemetryTarget{
		key:       strings.Join([]string{scheme, host, port}, "\x00"),
		targetURL: targetURL.String(),
		scheme:    scheme,
		host:      host,
		port:      port,
	}, true
}

func providerHostTelemetryDefaultPort(scheme string) string {
	switch scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func providerHostTelemetryView(target providerHostTelemetryTarget, point *providerHostTelemetryPoint) *managementv1.ProviderHostTelemetry {
	view := &managementv1.ProviderHostTelemetry{
		TargetUrl:    target.targetURL,
		Host:         target.host,
		Scheme:       target.scheme,
		Port:         target.port,
		Availability: managementv1.ProviderHostTelemetryAvailability_PROVIDER_HOST_TELEMETRY_AVAILABILITY_UNKNOWN,
		Reason:       "no_recent_sample",
	}
	if point == nil {
		return view
	}
	if point.latency != nil {
		view.LatencySeconds = *point.latency
	}
	if point.statusCode != nil {
		view.HttpStatusCode = int32(math.Round(*point.statusCode))
	}
	if point.sampledAt != nil {
		view.SampledAt = point.sampledAt
	}
	if point.success == nil {
		return view
	}
	if *point.success > 0 {
		view.Availability = managementv1.ProviderHostTelemetryAvailability_PROVIDER_HOST_TELEMETRY_AVAILABILITY_REACHABLE
		view.Reason = ""
		return view
	}
	view.Availability = managementv1.ProviderHostTelemetryAvailability_PROVIDER_HOST_TELEMETRY_AVAILABILITY_UNREACHABLE
	view.Reason = "probe_failed"
	return view
}

func sortedProviderHostTelemetry(items map[string]*managementv1.ProviderHostTelemetry) []*managementv1.ProviderHostTelemetry {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	out := make([]*managementv1.ProviderHostTelemetry, 0, len(keys))
	for _, key := range keys {
		out = append(out, cloneProviderHostTelemetry(items[key]))
	}
	return out
}

func cloneProviderHostTelemetry(item *managementv1.ProviderHostTelemetry) *managementv1.ProviderHostTelemetry {
	if item == nil {
		return nil
	}
	return proto.Clone(item).(*managementv1.ProviderHostTelemetry)
}
