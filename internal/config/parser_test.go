package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abhijeetydv/blast/internal/config"
)

func validYAML() string {
	return strings.Join([]string{
		`name: test-scenario`,
		`base_url: https://api.example.com`,
		`load:`,
		`  duration: 30s`,
		`  users: 1`,
		`steps:`,
		`  - name: step1`,
		`    method: GET`,
		`    path: /`,
	}, "\n")
}

func assertValidScenario(t *testing.T, got *config.ScenarioSpec) {
	if got.Name != "load-test" {
		t.Errorf("Name = %q, want %q", got.Name, "load-test")
	}
	if got.BaseURL != "https://httpbin.org" {
		t.Errorf("BaseURL = %q, want %q", got.BaseURL, "https://httpbin.org")
	}
	if got.Load.Duration != "60s" {
		t.Errorf("Duration = %q, want %q", got.Load.Duration, "60s")
	}
	if got.Load.DurationParsed != 60*time.Second {
		t.Errorf("DurationParsed = %v, want %v", got.Load.DurationParsed, 60*time.Second)
	}
	if got.Load.Users != 50 {
		t.Errorf("Users = %d, want %d", got.Load.Users, 50)
	}
	if got.Load.Rate != "100/s" {
		t.Errorf("Rate = %q, want %q", got.Load.Rate, "100/s")
	}
	if got.Load.RatePerSecond != 100.0 {
		t.Errorf("RatePerSecond = %f, want %f", got.Load.RatePerSecond, 100.0)
	}
	if !got.Options.CookieJar {
		t.Errorf("CookieJar = false, want true")
	}
	if got.Options.Timeout != "5s" {
		t.Errorf("Timeout = %q, want %q", got.Options.Timeout, "5s")
	}
	if got.Options.TimeoutParsed != 5*time.Second {
		t.Errorf("TimeoutParsed = %v, want %v", got.Options.TimeoutParsed, 5*time.Second)
	}
	if got.Options.MaxRedirects != 10 {
		t.Errorf("MaxRedirects = %d, want 10 (default)", got.Options.MaxRedirects)
	}
	if len(got.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(got.Steps))
	}
	if got.Steps[0].Name != "get-users" {
		t.Errorf("Steps[0].Name = %q, want %q", got.Steps[0].Name, "get-users")
	}
	if got.Steps[0].Method != "GET" {
		t.Errorf("Steps[0].Method = %q, want %q", got.Steps[0].Method, "GET")
	}
	if got.Steps[0].Path != "/users" {
		t.Errorf("Steps[0].Path = %q, want %q", got.Steps[0].Path, "/users")
	}
	if got.Steps[0].Timeout != "3s" {
		t.Errorf("Steps[0].Timeout = %q, want %q", got.Steps[0].Timeout, "3s")
	}
	if got.Steps[0].TimeoutParsed != 3*time.Second {
		t.Errorf("Steps[0].TimeoutParsed = %v, want %v", got.Steps[0].TimeoutParsed, 3*time.Second)
	}
	if got.Steps[0].Expect.Status != 200 {
		t.Errorf("Steps[0].Expect.Status = %d, want %d", got.Steps[0].Expect.Status, 200)
	}
	if got.Steps[0].Extract["id"] != "$.data.id" {
		t.Errorf("Steps[0].Extract[id] = %q, want %q", got.Steps[0].Extract["id"], "$.data.id")
	}
	if got.Thresholds.P99Ms != 500 {
		t.Errorf("Thresholds.P99Ms = %f, want %f", got.Thresholds.P99Ms, 500.0)
	}
	if got.Thresholds.P95Ms != 300 {
		t.Errorf("Thresholds.P95Ms = %f, want %f", got.Thresholds.P95Ms, 300.0)
	}
	if got.Thresholds.ErrorRate != 0.01 {
		t.Errorf("Thresholds.ErrorRate = %f, want %f", got.Thresholds.ErrorRate, 0.01)
	}
	if len(got.Thresholds.Steps) != 1 {
		t.Fatalf("len(Thresholds.Steps) = %d, want 1", len(got.Thresholds.Steps))
	}
	if got.Thresholds.Steps[0].Name != "get-users" {
		t.Errorf("Thresholds.Steps[0].Name = %q, want %q", got.Thresholds.Steps[0].Name, "get-users")
	}
	if got.Thresholds.Steps[0].P99Ms != 1000 {
		t.Errorf("Thresholds.Steps[0].P99Ms = %f, want %f", got.Thresholds.Steps[0].P99Ms, 1000.0)
	}
	if got.Regression.P99Percent != 10 {
		t.Errorf("Regression.P99Percent = %f, want %f", got.Regression.P99Percent, 10.0)
	}
	if got.Regression.ErrorRatePercent != 5 {
		t.Errorf("Regression.ErrorRatePercent = %f, want %f", got.Regression.ErrorRatePercent, 5.0)
	}
}

func TestLoad(t *testing.T) {
	// Note: no t.Parallel() here because Go 1.26's t.Setenv checks
	// the parent's parallel status and panics if set. Subtests that
	// do not call t.Setenv still use t.Parallel().

	tests := []struct {
		name         string
		yamlContent  string
		envVars      map[string]string
		wantErr      bool
		wantErrField string
		wantNonCfg   bool
		check        func(*testing.T, *config.ScenarioSpec)
		checkErr     func(*testing.T, error)
	}{
		// ─── Valid YAML ──────────────────────────────────────────────────────
		{
			name: "valid YAML file",
			yamlContent: strings.Join([]string{
				`name: load-test`,
				`base_url: https://httpbin.org`,
				`load:`,
				`  duration: 60s`,
				`  users: 50`,
				`  rate: 100/s`,
				`options:`,
				`  cookie_jar: true`,
				`  timeout: 5s`,
				`steps:`,
				`  - name: get-users`,
				`    method: GET`,
				`    path: /users`,
				`    timeout: 3s`,
				`    expect:`,
				`      status: 200`,
				`    extract:`,
				`      id: "$.data.id"`,
				`thresholds:`,
				`  p99_ms: 500`,
				`  p95_ms: 300`,
				`  error_rate: 0.01`,
				`  steps:`,
				`    - name: get-users`,
				`      p99_ms: 1000`,
				`regression:`,
				`  p99_percent: 10`,
				`  error_rate_percent: 5`,
			}, "\n"),
			check: assertValidScenario,
		},

		// ─── File errors ─────────────────────────────────────────────────────
		{
			name:       "file does not exist",
			wantErr:    true,
			wantNonCfg: true,
		},
		{
			name: "invalid YAML syntax",
			yamlContent: strings.Join([]string{
				`name: test`,
				`base_url: [invalid yaml`,
			}, "\n"),
			wantErr:    true,
			wantNonCfg: true,
		},

		// ─── ${VAR} interpolation ─────────────────────────────────────────────
		{
			name: "${VAR} interpolation when var is set",
			yamlContent: strings.NewReplacer(
				"test-scenario", "${TEST_SCENARIO_NAME}",
				"users: 1", "users: ${TEST_USERS}",
			).Replace(validYAML()),
			envVars: map[string]string{
				"TEST_SCENARIO_NAME": "interpolated-scenario",
				"TEST_USERS":         "7",
			},
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Name != "interpolated-scenario" {
					t.Errorf("Name = %q, want %q", got.Name, "interpolated-scenario")
				}
				if got.Load.Users != 7 {
					t.Errorf("Users = %d, want %d", got.Load.Users, 7)
				}
			},
		},
		{
			name: "${VAR} interpolation when var is NOT set",
			yamlContent: strings.NewReplacer(
				"test-scenario", "${UNDEFINED_VAR_X}",
			).Replace(validYAML()),
			wantErr:      true,
			wantErrField: "UNDEFINED_VAR_X",
		},
		{
			name: "${VAR:-default} when var unset",
			yamlContent: strings.NewReplacer(
				"test-scenario", "${SCENARIO_NAME:-default-name}",
				"users: 1", "users: ${NUM_USERS:-3}",
			).Replace(validYAML()),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Name != "default-name" {
					t.Errorf("Name = %q, want %q", got.Name, "default-name")
				}
				if got.Load.Users != 3 {
					t.Errorf("Users = %d, want %d", got.Load.Users, 3)
				}
			},
		},
		{
			name: "${VAR:-default} when var IS set",
			yamlContent: strings.NewReplacer(
				"test-scenario", "${OVERRIDE_NAME:-fallback}",
			).Replace(validYAML()),
			envVars: map[string]string{
				"OVERRIDE_NAME": "actual-value",
			},
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Name != "actual-value" {
					t.Errorf("Name = %q, want %q", got.Name, "actual-value")
				}
			},
		},
		{
			name: "multiple env vars in one file",
			yamlContent: strings.NewReplacer(
				"test-scenario", "${MULTI_NAME}",
				"https://api.example.com", "https://${MULTI_HOST}:${MULTI_PORT:-9090}",
				"users: 1", "users: ${MULTI_USERS:-1}",
			).Replace(validYAML()),
			envVars: map[string]string{
				"MULTI_NAME": "multi-test",
				"MULTI_HOST": "api.test.io",
			},
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Name != "multi-test" {
					t.Errorf("Name = %q, want %q", got.Name, "multi-test")
				}
				if got.BaseURL != "https://api.test.io:9090" {
					t.Errorf("BaseURL = %q, want %q", got.BaseURL, "https://api.test.io:9090")
				}
				if got.Load.Users != 1 {
					t.Errorf("Users = %d, want %d", got.Load.Users, 1)
				}
			},
		},
		{
			name: "missing closing brace",
			yamlContent: strings.NewReplacer(
				"test-scenario", "${UNCLOSED",
			).Replace(validYAML()),
			wantErr:    true,
			wantNonCfg: true,
			checkErr: func(t *testing.T, err error) {
				if !errors.Is(err, config.ErrMissingClosingBrace) {
					t.Errorf("expected ErrMissingClosingBrace, got %v", err)
				}
			},
		},
		{
			name: "valid YAML with env interpolation but validation fails",
			yamlContent: strings.NewReplacer(
				"test-scenario", "${EMPTY_NAME_VAR:-}",
			).Replace(validYAML()),
			wantErr:      true,
			wantErrField: "name",
		},
		{
			name: "${VAR:-} with empty default",
			yamlContent: strings.Join([]string{
				`name: test-scenario`,
				`base_url: https://api.example.com`,
				`load:`,
				`  duration: 30s`,
				`  users: 1`,
				`  rate: ${EMPTY_DEFAULT_RATE:-}`,
				`steps:`,
				`  - name: step1`,
				`    method: GET`,
				`    path: /`,
			}, "\n"),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Load.Rate != "" {
					t.Errorf("Rate = %q, want empty string", got.Load.Rate)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			// Go 1.24+ forbids using both t.Setenv and t.Parallel in
			// the same test. Subtests that don't need env vars can
			// still parallelize freely.
			if len(tt.envVars) == 0 {
				t.Parallel()
			}

			path := filepath.Join(t.TempDir(), "scenario.yaml")
			if tt.yamlContent != "" {
				if err := os.WriteFile(path, []byte(tt.yamlContent), 0644); err != nil {
					t.Fatal(err)
				}
			} else if tt.wantErr {
				path = filepath.Join(t.TempDir(), "nonexistent.yaml")
			}

			got, err := config.Load(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.wantErrField != "" {
					var ce *config.ConfigError
					if !errors.As(err, &ce) {
						t.Fatalf("expected *config.ConfigError, got %T: %v", err, err)
					}
					if ce.Field != tt.wantErrField {
						t.Errorf("wantErrField = %q, got %q", tt.wantErrField, ce.Field)
					}
				}
				if tt.wantNonCfg {
					var ce *config.ConfigError
					if errors.As(err, &ce) {
						t.Fatalf("expected non-ConfigError, got *config.ConfigError{Field: %q}", ce.Field)
					}
				}
				if tt.checkErr != nil {
					tt.checkErr(t, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
