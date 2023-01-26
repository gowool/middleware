package logger

import (
	"fmt"
	"github.com/gowool/wool"
	"golang.org/x/exp/slog"
	"regexp"
	"time"
)

type Config struct {
	ExcludeRegexStatus   string `mapstructure:"exclude_status"`
	ExcludeRegexMethod   string `mapstructure:"exclude_method"`
	ExcludeRegexEndpoint string `mapstructure:"exclude_endpoint"`
	IgnoreCtxLogger      bool   `mapstructure:"ignore_ctx_logger"`

	rxStatus   *regexp.Regexp
	rxMethod   *regexp.Regexp
	rxEndpoint *regexp.Regexp
}

func (cfg *Config) Init() {
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

type Logger struct {
	cfg *Config
	log *slog.Logger
}

func Middleware(cfg *Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg *Config) *Logger {
	cfg.Init()

	return &Logger{cfg: cfg, log: wool.Logger().WithGroup("middleware.logger")}
}

func (m *Logger) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		start := time.Now()
		err := next(c)
		end := time.Now()

		method := c.Req().Method
		endpoint := c.Req().URL.Path
		status := c.Res().Status()

		if !m.cfg.isOK(fmt.Sprintf("%d", status), method, endpoint) {
			return err
		}

		host := c.Req().URL.Host
		if host == "" {
			host = c.Req().Host
		}

		args := []any{
			"status", status,
			"method", method,
			"host", host,
			"path", endpoint,
			"query", c.Req().URL.RawQuery,
			"ip", c.Req().RemoteAddr,
			"user-agent", c.Req().UserAgent(),
			"duration", end.Sub(start),
			"latency", fmt.Sprintf("%s", end.Sub(start)),
		}

		if err != nil {
			m.log.Error(c.Req().URL.String(), err, args...)
		} else if status >= 400 {
			m.log.Warn(c.Req().URL.String(), args...)
		} else {
			m.log.Info(c.Req().URL.String(), args...)
		}

		return err
	}
}
