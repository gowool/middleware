package logger

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/gowool/wool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"regexp"
	"time"
)

const (
	formatInfo  = "%d [%s] latency=%s host=%s path=%s query=%s ip=%s user-agent=%s\n"
	formatError = "%d [%s] latency=%s host=%s path=%s query=%s ip=%s user-agent=%s error=%v\n"
)

type RequestLogger interface {
	Info(LogInfo)
	Warn(LogInfo)
	Error(LogInfo)
}

type LogInfo struct {
	Msg       string
	Status    int
	Method    string
	Host      string
	Path      string
	Query     string
	IP        string
	UserAgent string
	Latency   time.Duration
	Err       error
}

type Config struct {
	ExcludeRegexStatus   string `mapstructure:"exclude_status"`
	ExcludeRegexMethod   string `mapstructure:"exclude_method"`
	ExcludeRegexEndpoint string `mapstructure:"exclude_endpoint"`
	IgnoreCtxLogger      bool   `mapstructure:"ignore_ctx_logger"`
	Logger               RequestLogger

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
	if cfg.Logger == nil {
		cfg.Logger = newStdLogger()
	}
}

func (cfg *Config) isOK(status, method, endpoint string) bool {
	return (cfg.rxStatus == nil || !cfg.rxStatus.MatchString(status)) &&
		(cfg.rxMethod == nil || !cfg.rxMethod.MatchString(method)) &&
		(cfg.rxEndpoint == nil || !cfg.rxEndpoint.MatchString(endpoint))
}

type Logger struct {
	cfg    *Config
	logger RequestLogger
}

func Middleware(cfg *Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg *Config) *Logger {
	cfg.Init()

	return &Logger{cfg: cfg}
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

		li := LogInfo{
			Msg:       c.Req().URL.String(),
			Status:    status,
			Method:    method,
			Host:      host,
			Path:      endpoint,
			Query:     c.Req().URL.RawQuery,
			IP:        c.Req().RemoteAddr,
			UserAgent: c.Req().UserAgent(),
			Latency:   end.Sub(start),
			Err:       err,
		}

		if m.logger == nil {
			if m.cfg.IgnoreCtxLogger || c.Log() == nil {
				m.logger = m.cfg.Logger
			} else {
				m.logger = &zapLogger{logger: c.Log()}
			}
		}

		if err != nil {
			m.logger.Error(li)
		} else if status >= 400 {
			m.logger.Warn(li)
		} else {
			m.logger.Info(li)
		}

		return err
	}
}

func (li LogInfo) stdFields() []any {
	var fields = []any{
		li.Status,
		li.Method,
		li.Latency,
		li.Host,
		li.Path,
		li.Query,
		li.IP,
		li.UserAgent,
	}
	if li.Err != nil {
		fields = append(fields, li.Err)
	}
	return fields
}

func (li LogInfo) zapFields() []zapcore.Field {
	fields := []zapcore.Field{
		zap.Int("status", li.Status),
		zap.String("method", li.Method),
		zap.String("host", li.Host),
		zap.String("path", li.Path),
		zap.String("query", li.Query),
		zap.String("ip", li.IP),
		zap.String("user-agent", li.UserAgent),
		zap.Duration("duration", li.Latency),
		zap.String("latency", fmt.Sprintf("%s", li.Latency)),
	}
	if li.Err != nil {
		fields = append(fields, zap.Error(li.Err))
	}
	return fields
}

type stdLogger struct {
	prefixInfo  string
	prefixWarn  string
	prefixError string
	logger      *log.Logger
}

func newStdLogger() RequestLogger {
	return &stdLogger{
		prefixInfo:  color.New(color.FgHiGreen, color.Bold).Sprint("INFO "),
		prefixWarn:  color.New(color.FgHiYellow, color.Bold).Sprint("WARN "),
		prefixError: color.New(color.FgHiRed, color.Bold).Sprint("ERROR "),
		logger:      log.New(os.Stderr, "", log.Ldate|log.Lmicroseconds|log.Lshortfile),
	}
}

func (l *stdLogger) Info(i LogInfo) {
	l.logger.SetPrefix(l.prefixInfo)
	l.print(i)
}

func (l *stdLogger) Warn(i LogInfo) {
	l.logger.SetPrefix(l.prefixWarn)
	l.print(i)
}

func (l *stdLogger) Error(i LogInfo) {
	l.logger.SetPrefix(l.prefixError)
	l.print(i)
}

func (l *stdLogger) print(i LogInfo) {
	if i.Err == nil {
		l.logger.Printf(formatInfo, i.stdFields()...)
	} else {
		l.logger.Printf(formatError, i.stdFields()...)
	}
}

type zapLogger struct {
	logger *zap.Logger
}

func (l *zapLogger) Info(i LogInfo) {
	l.logger.Info(i.Msg, i.zapFields()...)
}

func (l *zapLogger) Warn(i LogInfo) {
	l.logger.Warn(i.Msg, i.zapFields()...)
}

func (l *zapLogger) Error(i LogInfo) {
	l.logger.Error(i.Msg, i.zapFields()...)
}
