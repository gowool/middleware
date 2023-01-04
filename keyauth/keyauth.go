package keyauth

import (
	"errors"
	"fmt"
	"github.com/gowool/wool"
)

// Validator defines a function to validate KeyAuth credentials.
type Validator func(c wool.Ctx, key string, source ExtractorSource) (bool, error)

// ErrorHandler defines a function which is executed for an invalid key.
type ErrorHandler func(c wool.Ctx, err error) error

type Config struct {
	// KeyLookup is a string in the form of "<source>:<name>" or "<source>:<name>,<source>:<name>" that is used
	// to extract key from the request.
	// Optional. Default value "header:Authorization".
	// Possible values:
	// - "header:<name>" or "header:<name>:<cut-prefix>"
	// 			`<cut-prefix>` is argument value to cut/trim prefix of the extracted value. This is useful if header
	//			value has static prefix like `Authorization: <auth-scheme> <authorisation-parameters>` where part that we
	//			want to cut is `<auth-scheme> ` note the space at the end.
	//			In case of basic authentication `Authorization: Basic <credentials>` prefix we want to remove is `Basic `.
	// - "query:<name>"
	// - "form:<name>"
	// - "cookie:<name>"
	// Multiple sources example:
	// - "header:Authorization,header:X-Api-Key"
	KeyLookup string `mapstructure:"key_lookup"`

	// ContinueOnIgnoredError allows the next middleware/handler to be called when ErrorHandler decides to
	// ignore the error (by returning `nil`).
	// This is useful when parts of your site/api allow public access and some authorized routes provide extra functionality.
	// In that case you can use ErrorHandler to set a default public key auth value in the request context
	// and continue. Some logic down the remaining execution chain needs to check that (public) key auth value then.
	ContinueOnIgnoredError bool `mapstructure:"continue_on_ignored_error"`

	// Validator is a function to validate key.
	// Required.
	Validator Validator

	// ErrorHandler defines a function which is executed for an invalid key.
	// It may be used to define a custom error.
	ErrorHandler ErrorHandler
}

func (cfg *Config) init() {
	if cfg.Validator == nil {
		panic(errors.New("key-auth middleware requires a validator function"))
	}
	if cfg.KeyLookup == "" {
		cfg.KeyLookup = "header:" + wool.HeaderAuthorization + ":Bearer "
	}
}

type KeyAuth struct {
	cfg        Config
	extractors []ValuesExtractor
}

func Middleware(cfg Config) wool.Middleware {
	return New(cfg).Middleware
}

func New(cfg Config) *KeyAuth {
	cfg.init()

	extractors, err := CreateExtractors(cfg.KeyLookup)
	if err != nil {
		panic(fmt.Errorf("key-auth middleware could not create key extractor: %w", err))
	}
	if len(extractors) == 0 {
		panic(errors.New("key-auth middleware could not create extractors from KeyLookup string"))
	}

	return &KeyAuth{cfg: cfg, extractors: extractors}
}

func (m *KeyAuth) Middleware(next wool.Handler) wool.Handler {
	return func(c wool.Ctx) error {
		var lastExtractorErr error
		var lastValidatorErr error
		for _, extractor := range m.extractors {
			keys, source, extrErr := extractor(c)
			if extrErr != nil {
				lastExtractorErr = extrErr
				continue
			}
			for _, key := range keys {
				valid, err := m.cfg.Validator(c, key, source)
				if err != nil {
					lastValidatorErr = err
					continue
				}
				if !valid {
					lastValidatorErr = wool.NewErrUnauthorized(nil, "invalid key")
					continue
				}
				return next(c)
			}
		}

		err := lastValidatorErr
		if err == nil {
			err = lastExtractorErr
		}
		if m.cfg.ErrorHandler != nil {
			tmpErr := m.cfg.ErrorHandler(c, err)
			if m.cfg.ContinueOnIgnoredError && tmpErr == nil {
				return next(c)
			}
			return tmpErr
		}

		if e, ok := err.(*wool.Error); ok {
			return e
		}

		if lastValidatorErr == nil {
			return wool.NewErrBadRequest(err, "missing key")
		}
		return wool.NewErrUnauthorized(err, "invalid key")
	}
}
