package cors

import (
	"github.com/gowool/wool"
	"net/http"
	"strconv"
	"strings"
)

var (
	allowCredentials = true

	DefaultConfig = Config{
		AllowedOrigin: "*",
		AllowedHeaders: strings.Join([]string{
			wool.HeaderContentType,
			wool.HeaderAccept,
			wool.HeaderAuthorization,
			wool.HeaderLastEventID,
		}, ","),
		AllowedMethods:   strings.Join(wool.DefaultMethods, ","),
		AllowCredentials: &allowCredentials,
		ExposedHeaders: strings.Join([]string{
			wool.HeaderContentType,
			wool.HeaderContentLanguage,
			wool.HeaderCacheControl,
			wool.HeaderConnection,
			wool.HeaderLocation,
			wool.HeaderLastModified,
			wool.HeaderExpires,
			wool.HeaderPragma,
		}, ","),
	}
)

type Config struct {
	// AllowedOrigin: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Origin
	AllowedOrigin string `mapstructure:"allowed_origin"`

	// AllowedHeaders: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers
	AllowedHeaders string `mapstructure:"allowed_headers"`

	// AllowedMethods: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Methods
	AllowedMethods string `mapstructure:"allowed_methods"`

	// AllowCredentials https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Credentials
	AllowCredentials *bool `mapstructure:"allow_credentials"`

	// ExposeHeaders:  https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Expose-Headers
	ExposedHeaders string `mapstructure:"exposed_headers"`

	// MaxAge of CORS headers in seconds/
	MaxAge int `mapstructure:"max_age"`
}

type CORS struct {
	cfg Config
}

func Middleware(cfg Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg Config) *CORS {
	return &CORS{cfg: cfg}
}

func (m *CORS) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		headers := c.Res().Header()

		headers.Add(wool.HeaderVary, "Origin")

		if m.cfg.AllowedOrigin != "" {
			headers.Set(wool.HeaderAccessControlAllowOrigin, m.cfg.AllowedOrigin)
		}

		if m.cfg.AllowedHeaders != "" {
			headers.Set(wool.HeaderAccessControlAllowHeaders, m.cfg.AllowedHeaders)
		}

		if m.cfg.AllowCredentials != nil {
			headers.Set(wool.HeaderAccessControlAllowCredentials, strconv.FormatBool(*m.cfg.AllowCredentials))
		}

		if c.Req().Method == http.MethodOptions {
			headers.Add(wool.HeaderVary, "Access-Control-Request-Method")
			headers.Add(wool.HeaderVary, "Access-Control-Request-Headers")

			if m.cfg.AllowedMethods != "" {
				headers.Set(wool.HeaderAccessControlAllowMethods, m.cfg.AllowedMethods)
			}

			if m.cfg.MaxAge > 0 {
				headers.Set(wool.HeaderAccessControlMaxAge, strconv.Itoa(m.cfg.MaxAge))
			}

			return c.OK()
		}

		if m.cfg.ExposedHeaders != "" {
			headers.Set(wool.HeaderAccessControlExposeHeaders, m.cfg.ExposedHeaders)
		}

		return next(c)
	}
}
