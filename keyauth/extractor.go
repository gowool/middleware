package keyauth

import (
	"fmt"
	"github.com/gowool/wool"
	"github.com/spf13/cast"
	"net/textproto"
	"strings"
)

const extractorLimit = 20

type ExtractorSource string

const (
	ExtractorSourceHeader ExtractorSource = "header"
	ExtractorSourceQuery  ExtractorSource = "query"
	ExtractorSourcePath   ExtractorSource = "path"
	ExtractorSourceForm   ExtractorSource = "form"
	ExtractorSourceCtx    ExtractorSource = "ctx"
)

type ValueExtractorError struct {
	message string
}

func (e *ValueExtractorError) Error() string {
	return e.message
}

var (
	ErrHeaderExtractorValueMissing = &ValueExtractorError{message: "missing value in request header"}
	ErrHeaderExtractorValueInvalid = &ValueExtractorError{message: "invalid value in request header"}
	ErrQueryExtractorValueMissing  = &ValueExtractorError{message: "missing value in query string"}
	ErrPathExtractorValueMissing   = &ValueExtractorError{message: "missing value in path params"}
	ErrFormExtractorValueMissing   = &ValueExtractorError{message: "missing value in form"}
	ErrCtxExtractorValueMissing    = &ValueExtractorError{message: "missing value in ctx"}
)

type ValuesExtractor func(c wool.Ctx) ([]string, ExtractorSource, error)

func CreateExtractors(lookups string) ([]ValuesExtractor, error) {
	if lookups == "" {
		return nil, nil
	}
	sources := strings.Split(lookups, ",")
	var extractors = make([]ValuesExtractor, 0)
	for _, source := range sources {
		parts := strings.Split(source, ":")
		if len(parts) < 2 {
			return nil, fmt.Errorf("extractor source for lookup could not be split into needed parts: %v", source)
		}

		switch parts[0] {
		case "query":
			extractors = append(extractors, valuesFromQuery(parts[1]))
		case "path":
			extractors = append(extractors, valuesFromPath(parts[1]))
		case "ctx":
			extractors = append(extractors, valuesFromCtx(parts[1]))
		case "form":
			extractors = append(extractors, valuesFromForm(parts[1]))
		case "header":
			prefix := ""
			if len(parts) > 2 {
				prefix = parts[2]
			}
			extractors = append(extractors, valuesFromHeader(parts[1], prefix))
		}
	}
	return extractors, nil
}

func valuesFromHeader(header string, valuePrefix string) ValuesExtractor {
	prefixLen := len(valuePrefix)
	header = textproto.CanonicalMIMEHeaderKey(header)
	return func(c wool.Ctx) ([]string, ExtractorSource, error) {
		values := c.Req().Header.Values(header)
		if len(values) == 0 {
			return nil, ExtractorSourceHeader, ErrHeaderExtractorValueMissing
		}

		var result []string
		for i, value := range values {
			if prefixLen == 0 {
				result = append(result, value)
				if i >= extractorLimit-1 {
					break
				}
				continue
			}
			if len(value) > prefixLen && strings.EqualFold(value[:prefixLen], valuePrefix) {
				result = append(result, value[prefixLen:])
				if i >= extractorLimit-1 {
					break
				}
			}
		}

		if len(result) == 0 {
			if prefixLen > 0 {
				return nil, ExtractorSourceHeader, ErrHeaderExtractorValueInvalid
			}
			return nil, ExtractorSourceHeader, ErrHeaderExtractorValueMissing
		}
		return result, ExtractorSourceHeader, nil
	}
}

func valuesFromQuery(param string) ValuesExtractor {
	return func(c wool.Ctx) ([]string, ExtractorSource, error) {
		if result := valuesFrom(c.Req().QueryParams(), param); result != nil {
			return result, ExtractorSourceQuery, nil
		}
		return nil, ExtractorSourceQuery, ErrQueryExtractorValueMissing
	}
}

func valuesFromPath(param string) ValuesExtractor {
	return func(c wool.Ctx) ([]string, ExtractorSource, error) {
		if result := valuesFrom(c.Req().PathParams(), param); result != nil {
			return result, ExtractorSourcePath, nil
		}
		return nil, ExtractorSourcePath, ErrPathExtractorValueMissing
	}
}

func valuesFromCtx(name string) ValuesExtractor {
	return func(c wool.Ctx) ([]string, ExtractorSource, error) {
		if data, err := cast.ToStringMapStringSliceE(c.Store()); err == nil {
			if result := valuesFrom(data, name); result != nil {
				return result, ExtractorSourceCtx, nil
			}
		}
		return nil, ExtractorSourceCtx, ErrCtxExtractorValueMissing
	}
}

func valuesFromForm(name string) ValuesExtractor {
	return func(c wool.Ctx) ([]string, ExtractorSource, error) {
		if data, err := c.Req().FormValues(); err == nil {
			if result := valuesFrom(data, name); result != nil {
				return result, ExtractorSourceForm, nil
			}
		}
		return nil, ExtractorSourceForm, ErrFormExtractorValueMissing
	}
}

func valuesFrom(data map[string][]string, name string) []string {
	if data != nil {
		result := data[name]
		if l := len(result); l > 0 {
			if l >= extractorLimit {
				result = result[:extractorLimit]
			}
			v := make([]string, 0, len(result))
			return append(v, result...)
		}
	}
	return nil
}
