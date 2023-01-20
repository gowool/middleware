package prometheus

import (
	"fmt"
	"github.com/gowool/wool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"regexp"
	"time"
)

func Mount(w *wool.Wool) {
	w.Get("/metrics", wool.ToHandler(promhttp.Handler()))
}

var labels = []string{"version", "status", "endpoint", "method"}

type RequestLabelMapping func(wool.Ctx) string

type Config struct {
	Version              string `mapstructure:"version"`
	Namespace            string `mapstructure:"namespace"`
	ExcludeRegexStatus   string `mapstructure:"exclude_status"`
	ExcludeRegexMethod   string `mapstructure:"exclude_method"`
	ExcludeRegexEndpoint string `mapstructure:"exclude_endpoint"`
	EndpointLabelMapping RequestLabelMapping

	rxStatus   *regexp.Regexp
	rxMethod   *regexp.Regexp
	rxEndpoint *regexp.Regexp
}

func (cfg *Config) Init() {
	if cfg.Version == "" {
		cfg.Version = "(untracked)"
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "wool"
	}

	if cfg.EndpointLabelMapping == nil {
		cfg.EndpointLabelMapping = func(c wool.Ctx) string {
			return c.Req().URL.Path
		}
	}
	if cfg.ExcludeRegexStatus != "" {
		cfg.rxStatus, _ = regexp.Compile(cfg.ExcludeRegexStatus)
	}
	if cfg.ExcludeRegexMethod != "" {
		cfg.rxMethod, _ = regexp.Compile(cfg.ExcludeRegexMethod)
	}
	if cfg.ExcludeRegexEndpoint != "" {
		cfg.rxEndpoint, _ = regexp.Compile(cfg.ExcludeRegexEndpoint)
	}
}

func (cfg *Config) isOK(status, method, endpoint string) bool {
	return (cfg.rxStatus == nil || !cfg.rxStatus.MatchString(status)) &&
		(cfg.rxMethod == nil || !cfg.rxMethod.MatchString(method)) &&
		(cfg.rxEndpoint == nil || !cfg.rxEndpoint.MatchString(endpoint))
}

type Prometheus struct {
	cfg           *Config
	uptime        *prometheus.CounterVec
	reqCount      *prometheus.CounterVec
	reqDuration   *prometheus.HistogramVec
	reqSizeBytes  *prometheus.SummaryVec
	respSizeBytes *prometheus.SummaryVec
}

func Middleware(cfg *Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg *Config) *Prometheus {
	cfg.Init()

	m := &Prometheus{
		cfg: cfg,
		uptime: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "uptime",
				Help:      "HTTP uptime.",
			}, nil,
		),
		reqCount: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Name:      "http_request_count_total",
				Help:      "Total number of HTTP requests made.",
			}, labels,
		),
		reqDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latencies in seconds.",
			}, labels,
		),
		reqSizeBytes: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: cfg.Namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request sizes in bytes.",
			}, labels,
		),
		respSizeBytes: prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Namespace: cfg.Namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response sizes in bytes.",
			}, labels,
		),
	}

	prometheus.MustRegister(m.uptime, m.reqCount, m.reqDuration, m.reqSizeBytes, m.respSizeBytes)

	go m.recordUptime()

	return m
}

func (m *Prometheus) recordUptime() {
	for range time.Tick(time.Second) {
		m.uptime.WithLabelValues().Inc()
	}
}

func (m *Prometheus) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		start := time.Now()

		err := next(c)

		status := fmt.Sprintf("%d", c.Res().Status())
		method := c.Req().Method
		endpoint := m.cfg.EndpointLabelMapping(c)

		if !m.cfg.isOK(status, method, endpoint) {
			return err
		}

		lvs := []string{m.cfg.Version, status, endpoint, method}

		m.reqCount.WithLabelValues(lvs...).Inc()
		m.reqDuration.WithLabelValues(lvs...).Observe(time.Since(start).Seconds())
		m.reqSizeBytes.WithLabelValues(lvs...).Observe(calcRequestSize(c.Req().Request))
		m.respSizeBytes.WithLabelValues(lvs...).Observe(float64(c.Res().Size()))

		return err
	}
}

func calcRequestSize(r *http.Request) float64 {
	size := 0
	if r.URL != nil {
		size = len(r.URL.String())
	}

	size += len(r.Method)
	size += len(r.Proto)

	for name, values := range r.Header {
		size += len(name)
		for _, value := range values {
			size += len(value)
		}
	}
	size += len(r.Host)

	// r.Form and r.MultipartForm are assumed to be included in r.URL.
	if r.ContentLength != -1 {
		size += int(r.ContentLength)
	}
	return float64(size)
}
