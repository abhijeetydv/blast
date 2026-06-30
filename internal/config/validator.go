package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error: %s: %s", e.Field, e.Message)
}

// Validate checks every field of spec and populates all parsed fields
func Validate(spec *ScenarioSpec) error {
	if err := validateName(spec); err != nil {
		return err
	}
	if err := validateBaseURL(spec); err != nil {
		return err
	}
	if err := validateLoad(&spec.Load); err != nil {
		return err
	}
	if err := validateSteps(spec.Steps); err != nil {
		return err
	}
	if err := validateOptions(&spec.Options); err != nil {
		return err
	}
	return nil
}

func validateName(spec *ScenarioSpec) error {
	if strings.TrimSpace(spec.Name) == "" {
		return &ConfigError{Field: "name", Message: "must be a non-empty string"}
	}
	return nil
}

func validateBaseURL(spec *ScenarioSpec) error {
	if strings.TrimSpace(spec.BaseURL) == "" {
		return &ConfigError{Field: "base_url", Message: "must be a non-empty string"}
	}
	u, err := url.Parse(spec.BaseURL)
	if err != nil {
		return &ConfigError{Field: "base_url", Message: fmt.Sprintf("%q is not a valid URL: %v", spec.BaseURL, err)}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return &ConfigError{Field: "base_url", Message: fmt.Sprintf("scheme must be http or https, got %q", u.Scheme)}
	}
	return nil
}

func validateLoad(l *LoadSpec) error {

	d, err := time.ParseDuration(l.Duration)
	if err != nil {
		return &ConfigError{Field: "load.duration", Message: fmt.Sprintf("%q is not a valid duration: %v", l.Duration, err)}
	}
	if d <= 0 {
		return &ConfigError{Field: "load.duration", Message: fmt.Sprintf("must be greater than 0, got %s", d)}
	}
	l.DurationParsed = d

	if l.Users != 0 && l.Users < 1 {
		return &ConfigError{Field: "load.users", Message: fmt.Sprintf("if set, must be > 0, got %d", l.Users)}
	}

	if l.Rate != "" {
		rps, err := parseRate(l.Rate)
		if err != nil {
			return &ConfigError{Field: "load.rate", Message: err.Error()}
		}
		l.RatePerSecond = rps
	}

	if l.Users == 0 && l.Rate == "" {
		return &ConfigError{Field: "load", Message: "at least one of users or rate must be set"}
	}

	if l.Ramp != nil {
		if err := validateRamp(l.Ramp); err != nil {
			return err
		}
	}

	return nil
}

func parseRate(raw string) (float64, error) {
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("%q must be in the format N/s or N/m", raw)
	}

	n, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%q: N must be a number > 0", raw)
	}

	unit := strings.TrimSpace(parts[1])
	switch unit {
	case "s":
		return n, nil
	case "m":
		return n / 60.0, nil
	default:
		return 0, fmt.Errorf("%q: unit must be 's' or 'm', got %q", raw, unit)
	}
}

func validateRamp(r *RampSpec) error {
	if strings.TrimSpace(r.From) == "" {
		return &ConfigError{Field: "load.ramp.from", Message: "required when ramp is set"}
	}
	if strings.TrimSpace(r.To) == "" {
		return &ConfigError{Field: "load.ramp.to", Message: "required when ramp is set"}
	}
	if strings.TrimSpace(r.Over) == "" {
		return &ConfigError{Field: "load.ramp.over", Message: "required when ramp is set"}
	}

	fromRPS, err := parseRate(r.From)
	if err != nil {
		return &ConfigError{Field: "load.ramp.from", Message: err.Error()}
	}
	r.FromRPS = fromRPS

	toRPS, err := parseRate(r.To)
	if err != nil {
		return &ConfigError{Field: "load.ramp.to", Message: err.Error()}
	}
	r.ToRPS = toRPS

	over, err := time.ParseDuration(r.Over)
	if err != nil {
		return &ConfigError{Field: "load.ramp.over", Message: fmt.Sprintf("%q is not a valid duration: %v", r.Over, err)}
	}
	if over <= 0 {
		return &ConfigError{Field: "load.ramp.over", Message: fmt.Sprintf("must be greater than 0, got %s", over)}
	}
	r.OverParsed = over

	return nil
}

func validateSteps(steps []StepSpec) error {
	if len(steps) == 0 {
		return &ConfigError{Field: "steps", Message: "at least one step is required"}
	}

	seen := make(map[string]bool, len(steps))

	for i := range steps {
		s := &steps[i]
		prefix := fmt.Sprintf("steps[%d]", i)

		if strings.TrimSpace(s.Name) == "" {
			return &ConfigError{Field: prefix + ".name", Message: "must be a non-empty string"}
		}
		if seen[s.Name] {
			return &ConfigError{Field: prefix + ".name", Message: fmt.Sprintf("%q is a duplicate step name", s.Name)}
		}
		seen[s.Name] = true

		if err := validateMethod(s, prefix); err != nil {
			return err
		}

		if strings.TrimSpace(s.Path) == "" {
			return &ConfigError{Field: prefix + ".path", Message: "must be a non-empty string"}
		}

		if !strings.HasPrefix(s.Path, "/") {
			return &ConfigError{Field: prefix + ".path", Message: fmt.Sprintf("must start with '/', got %q", s.Path)}
		}

		if err := validateBodyExclusivity(s, prefix); err != nil {
			return err
		}

		if err := validateExpect(&s.Expect, prefix); err != nil {
			return err
		}

		if err := validateExtract(s.Extract, prefix); err != nil {
			return err
		}

		if s.Timeout != "" {
			d, err := time.ParseDuration(s.Timeout)
			if err != nil {
				return &ConfigError{Field: prefix + ".timeout", Message: fmt.Sprintf("%q is not a valid duration: %v", s.Timeout, err)}
			}
			s.TimeoutParsed = d
		}
	}

	return nil
}

var allowedMethods = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
	"HEAD":   true,
}

func validateMethod(s *StepSpec, prefix string) error {
	upper := strings.ToUpper(strings.TrimSpace(s.Method))
	if !allowedMethods[upper] {
		return &ConfigError{Field: prefix + ".method", Message: fmt.Sprintf("must be one of GET, POST, PUT, PATCH, DELETE, HEAD; got %q", s.Method)}
	}
	s.Method = upper
	return nil
}

func validateBodyExclusivity(s *StepSpec, prefix string) error {
	count := 0
	if s.Body != nil {
		count++
	}
	if s.BodyRaw != "" {
		count++
	}
	if len(s.BodyForm) > 0 {
		count++
	}
	if count > 1 {
		return &ConfigError{Field: prefix + ".body", Message: "at most one of body, body_raw, or body_form may be set"}
	}
	return nil
}

func validateExpect(e *ExpectSpec, prefix string) error {
	hasStatus := e.Status != 0
	hasRange := e.StatusRange != [2]int{}
	if hasStatus && hasRange {
		return &ConfigError{Field: prefix + ".expect", Message: "at most one of status or status_range may be set"}
	}
	return nil
}

func validateExtract(ext map[string]string, prefix string) error {
	for key, val := range ext {
		if !strings.HasPrefix(val, "$.") &&
			!strings.HasPrefix(val, "regex:") &&
			!strings.HasPrefix(val, "header:") {
			return &ConfigError{
				Field:   fmt.Sprintf("%s.extract[%s]", prefix, key),
				Message: fmt.Sprintf("value must begin with \"$.\", \"regex:\", or \"header:\"; got %q", val),
			}
		}
	}
	return nil
}

func validateOptions(o *OptionsSpec) error {
	if o.Timeout != "" {
		d, err := time.ParseDuration(o.Timeout)
		if err != nil {
			return &ConfigError{Field: "options.timeout", Message: fmt.Sprintf("%q is not a valid duration: %v", o.Timeout, err)}
		}
		o.TimeoutParsed = d
	}

	if o.MaxRedirects == 0 {
		o.MaxRedirects = 10
	}

	return nil
}