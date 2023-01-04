package secure

import (
	"fmt"
	"github.com/gowool/wool"
)

type Config struct {
	// XSSProtection provides protection against cross-site scripting attack (XSS)
	// by setting the `X-XSS-Protection` header.
	// Optional. Default value "1; mode=block".
	XSSProtection string `mapstructure:"xss_protection"`

	// ContentTypeNosniff provides protection against overriding Content-Type
	// header by setting the `X-Content-Type-Options` header.
	// Optional. Default value "nosniff".
	ContentTypeNosniff string `mapstructure:"content_type_nosniff"`

	// XFrameOptions can be used to indicate whether or not a browser should
	// be allowed to render a page in a <frame>, <iframe> or <object> .
	// Sites can use this to avoid clickjacking attacks, by ensuring that their
	// content is not embedded into other sites.provides protection against
	// clickjacking.
	// Optional. Default value "SAMEORIGIN".
	// Possible values:
	// - "SAMEORIGIN" - The page can only be displayed in a frame on the same origin as the page itself.
	// - "DENY" - The page cannot be displayed in a frame, regardless of the site attempting to do so.
	// - "ALLOW-FROM uri" - The page can only be displayed in a frame on the specified origin.
	XFrameOptions string `mapstructure:"x_frame_options"`

	// HSTSMaxAge sets the `Strict-Transport-Security` header to indicate how
	// long (in seconds) browsers should remember that this site is only to
	// be accessed using HTTPS. This reduces your exposure to some SSL-stripping
	// man-in-the-middle (MITM) attacks.
	// Optional. Default value 0.
	HSTSMaxAge int `mapstructure:"hsts_max_age"`

	// HSTSExcludeSubdomains won't include subdomains tag in the `Strict Transport Security`
	// header, excluding all subdomains from security policy. It has no effect
	// unless HSTSMaxAge is set to a non-zero value.
	// Optional. Default value false.
	HSTSExcludeSubdomains bool `mapstructure:"hsts_exclude_subdomains"`

	// ContentSecurityPolicy sets the `Content-Security-Policy` header providing
	// security against cross-site scripting (XSS), clickjacking and other code
	// injection attacks resulting from execution of malicious content in the
	// trusted web page context.
	// Optional. Default value "".
	ContentSecurityPolicy string `mapstructure:"content_security_policy"`

	// CSPReportOnly would use the `Content-Security-Policy-Report-Only` header instead
	// of the `Content-Security-Policy` header. This allows iterative updates of the
	// content security policy by only reporting the violations that would
	// have occurred instead of blocking the resource.
	// Optional. Default value false.
	CSPReportOnly bool `mapstructure:"csp_report_only"`

	// HSTSPreloadEnabled will add the preload tag in the `Strict Transport Security`
	// header, which enables the domain to be included in the HSTS preload list
	// maintained by Chrome (and used by Firefox and Safari): https://hstspreload.org/
	// Optional.  Default value false.
	HSTSPreloadEnabled bool `mapstructure:"hsts_preload_enabled"`

	// ReferrerPolicy sets the `Referrer-Policy` header providing security against
	// leaking potentially sensitive request paths to third parties.
	// Optional. Default value "".
	ReferrerPolicy string `mapstructure:"referrer_policy"`
}

func (cfg *Config) init() {
	if cfg.XSSProtection == "" {
		cfg.XSSProtection = "1; mode=block"
	}
	if cfg.ContentTypeNosniff == "" {
		cfg.ContentTypeNosniff = "nosniff"
	}
	if cfg.XFrameOptions == "" {
		cfg.XFrameOptions = "SAMEORIGIN"
	}
}

type Secure struct {
	cfg Config
}

func Middleware(cfg Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg Config) *Secure {
	cfg.init()

	return &Secure{cfg: cfg}
}

func (m *Secure) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		if m.cfg.XSSProtection != "" {
			c.Res().Header().Set(wool.HeaderXXSSProtection, m.cfg.XSSProtection)
		}
		if m.cfg.ContentTypeNosniff != "" {
			c.Res().Header().Set(wool.HeaderXContentTypeOptions, m.cfg.ContentTypeNosniff)
		}
		if m.cfg.XFrameOptions != "" {
			c.Res().Header().Set(wool.HeaderXFrameOptions, m.cfg.XFrameOptions)
		}
		if (c.Req().IsTLS() || c.Req().Header.Get(wool.HeaderXForwardedProto) == "https") && m.cfg.HSTSMaxAge != 0 {
			subdomains := ""
			if !m.cfg.HSTSExcludeSubdomains {
				subdomains = "; includeSubdomains"
			}
			if m.cfg.HSTSPreloadEnabled {
				subdomains = fmt.Sprintf("%s; preload", subdomains)
			}
			c.Res().Header().Set(wool.HeaderStrictTransportSecurity, fmt.Sprintf("max-age=%d%s", m.cfg.HSTSMaxAge, subdomains))
		}
		if m.cfg.ContentSecurityPolicy != "" {
			if m.cfg.CSPReportOnly {
				c.Res().Header().Set(wool.HeaderContentSecurityPolicyReportOnly, m.cfg.ContentSecurityPolicy)
			} else {
				c.Res().Header().Set(wool.HeaderContentSecurityPolicy, m.cfg.ContentSecurityPolicy)
			}
		}
		if m.cfg.ReferrerPolicy != "" {
			c.Res().Header().Set(wool.HeaderReferrerPolicy, m.cfg.ReferrerPolicy)
		}
		return next(c)
	}
}
