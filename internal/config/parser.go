package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrMissingClosingBrace = errors.New("missing closing brace in variable reference")

func Load(path string) (*ScenarioSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading scenario file: %w", err)
	}

	interpolated, err := interpolateEnv(data)
	if err != nil {
		return nil, err
	}

	var spec ScenarioSpec
	if err := yaml.Unmarshal(interpolated, &spec); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := Validate(&spec); err != nil {
		return nil, err
	}
	return &spec, nil
}

// interpolateEnv performs a single-pass scan over raw, replacing every
// ${VAR_NAME} or ${VAR_NAME:-default} token with the corresponding
// environment variable value.
//
// Rules:
//   - ${VAR}          → os.Getenv("VAR"); error if unset/empty.
//   - ${VAR:-default} → os.Getenv("VAR") or "default" if unset/empty.
//
// The scanner advances character-by-character so that substituted values
// are never re-scanned, which prevents double-substitution bugs.
func interpolateEnv(raw []byte) ([]byte, error) {
	var buf strings.Builder
	buf.Grow(len(raw))

	for i := 0; i < len(raw); i++ {
		
		if raw[i] == '$' && i+1 < len(raw) && raw[i+1] == '{' {
			
			j := i + 2
			for j < len(raw) && raw[j] != '}' {
				j++
			}
			if j >= len(raw) {
				return nil, fmt.Errorf("%w: at byte offset %d", ErrMissingClosingBrace, i)
			}

			expr := string(raw[i+2 : j])

			varName, defaultVal, hasDefault := parseVarExpr(expr)

			value, isSet := os.LookupEnv(varName)
			switch {
			case isSet:
				buf.WriteString(value)
			case hasDefault:
				buf.WriteString(defaultVal)
			default:
				return nil, &ConfigError{
					Field:   varName,
					Message: fmt.Sprintf("required environment variable %q is not set", varName),
				}
			}

			i = j
			continue
		}
		buf.WriteByte(raw[i])
	}

	return []byte(buf.String()), nil
}

// parseVarExpr splits a variable expression into its name, optional default
// value, and a flag indicating whether a default was specified.
//
// Examples:
//
//	"HOST"          → ("HOST", "",      false)
//	"HOST:-localhost" → ("HOST", "localhost", true)
//	"PORT:-"        → ("PORT", "",      true)   // explicit empty default
func parseVarExpr(expr string) (name, defaultVal string, hasDefault bool) {

	if idx := strings.Index(expr, ":-"); idx >= 0 {
		return expr[:idx], expr[idx+2:], true
	}
	return expr, "", false
}
