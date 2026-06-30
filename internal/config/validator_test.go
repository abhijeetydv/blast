package config_test

import (
	"errors"
	"testing"
	"time"

	"github.com/abhijeetydv/blast/internal/config"
)

func validSpec() *config.ScenarioSpec {
	return &config.ScenarioSpec{
		Name:    "test",
		BaseURL: "https://example.com",
		Load:    config.LoadSpec{Duration: "30s", Users: 1},
		Steps:   []config.StepSpec{{Name: "s1", Method: "GET", Path: "/"}},
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spec         *config.ScenarioSpec
		wantErr      bool
		wantErrField string
		check        func(*testing.T, *config.ScenarioSpec)
	}{
		// ─── Valid minimal spec ───────────────────────────────────────────────
		{
			name: "valid minimal spec",
			spec: validSpec(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Load.DurationParsed != 30*time.Second {
					t.Errorf("DurationParsed = %v, want %v", got.Load.DurationParsed, 30*time.Second)
				}
				if got.Options.MaxRedirects != 10 {
					t.Errorf("MaxRedirects = %d, want 10 (default)", got.Options.MaxRedirects)
				}
			},
		},

		// ─── Name ─────────────────────────────────────────────────────────────
		{
			name: "missing name",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Name = ""; return s }(),
			wantErr: true, wantErrField: "name",
		},
		{
			name: "empty name (whitespace only)",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Name = "   "; return s }(),
			wantErr: true, wantErrField: "name",
		},

		// ─── Base URL ─────────────────────────────────────────────────────────
		{
			name: "missing base_url",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.BaseURL = ""; return s }(),
			wantErr: true, wantErrField: "base_url",
		},
		{
			name: "base_url with no scheme",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.BaseURL = "example.com"; return s }(),
			wantErr: true, wantErrField: "base_url",
		},
		{
			name: "base_url with invalid characters",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.BaseURL = "http://invalid url with spaces"; return s }(),
			wantErr: true, wantErrField: "base_url",
		},
		{
			name: "base_url with ftp:// scheme",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.BaseURL = "ftp://example.com"; return s }(),
			wantErr: true, wantErrField: "base_url",
		},

		// ─── Load duration ────────────────────────────────────────────────────
		{
			name: "load.duration missing",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Duration = ""; return s }(),
			wantErr: true, wantErrField: "load.duration",
		},
		{
			name: "load.duration not parseable",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Duration = "abc"; return s }(),
			wantErr: true, wantErrField: "load.duration",
		},
		{
			name: "load.duration zero",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Duration = "0s"; return s }(),
			wantErr: true, wantErrField: "load.duration",
		},

		// ─── Load users / rate ────────────────────────────────────────────────
		{
			name: "load.users negative",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Users = -5; return s }(),
			wantErr: true, wantErrField: "load.users",
		},
		{
			name: "load with neither users nor rate set",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Users = 0; return s }(),
			wantErr: true, wantErrField: "load",
		},
		{
			name: "load with rate only",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Users = 0; s.Load.Rate = "100/s"; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Load.RatePerSecond != 100.0 {
					t.Errorf("RatePerSecond = %f, want 100.0", got.Load.RatePerSecond)
				}
			},
		},
		{
			name: "load with users and rate",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Users = 5; s.Load.Rate = "50/s"; return s }(),
		},

		// ─── Load rate format ─────────────────────────────────────────────────
		{
			name: "load.rate bad format",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Users = 0; s.Load.Rate = "100rps"; return s }(),
			wantErr: true, wantErrField: "load.rate",
		},
		{
			name: "load.rate zero",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Users = 0; s.Load.Rate = "0/s"; return s }(),
			wantErr: true, wantErrField: "load.rate",
		},
		{
			name: "load.rate bad unit",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Users = 0; s.Load.Rate = "100/h"; return s }(),
			wantErr: true, wantErrField: "load.rate",
		},
		{
			name: "load.rate valid 100/s",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Rate = "100/s"; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Load.RatePerSecond != 100.0 {
					t.Errorf("RatePerSecond = %f, want 100.0", got.Load.RatePerSecond)
				}
			},
		},
		{
			name: "load.rate valid 60/m",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Rate = "60/m"; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Load.RatePerSecond != 1.0 {
					t.Errorf("RatePerSecond = %f, want 1.0", got.Load.RatePerSecond)
				}
			},
		},

		// ─── Load ramp ────────────────────────────────────────────────────────
		{
			name: "load.ramp.from missing",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Ramp = &config.RampSpec{To: "100/s", Over: "30s"}; return s }(),
			wantErr: true, wantErrField: "load.ramp.from",
		},
		{
			name: "load.ramp.to missing",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Ramp = &config.RampSpec{From: "10/s", Over: "30s"}; return s }(),
			wantErr: true, wantErrField: "load.ramp.to",
		},
		{
			name: "load.ramp.over missing",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Ramp = &config.RampSpec{From: "10/s", To: "100/s"}; return s }(),
			wantErr: true, wantErrField: "load.ramp.over",
		},
		{
			name: "load.ramp.from invalid rate",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Ramp = &config.RampSpec{From: "bad", To: "100/s", Over: "30s"}; return s }(),
			wantErr: true, wantErrField: "load.ramp.from",
		},
		{
			name: "load.ramp.to invalid rate",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Ramp = &config.RampSpec{From: "10/s", To: "bad", Over: "30s"}; return s }(),
			wantErr: true, wantErrField: "load.ramp.to",
		},
		{
			name: "load.ramp.over invalid duration",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Ramp = &config.RampSpec{From: "10/s", To: "100/s", Over: "not-a-duration"}; return s }(),
			wantErr: true, wantErrField: "load.ramp.over",
		},
		{
			name: "load.ramp.over zero",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Ramp = &config.RampSpec{From: "10/s", To: "100/s", Over: "0s"}; return s }(),
			wantErr: true, wantErrField: "load.ramp.over",
		},
		{
			name: "load.ramp valid",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Load.Ramp = &config.RampSpec{From: "10/s", To: "200/s", Over: "30s"}; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Load.Ramp.FromRPS != 10.0 {
					t.Errorf("FromRPS = %f, want 10.0", got.Load.Ramp.FromRPS)
				}
				if got.Load.Ramp.ToRPS != 200.0 {
					t.Errorf("ToRPS = %f, want 200.0", got.Load.Ramp.ToRPS)
				}
				if got.Load.Ramp.OverParsed != 30*time.Second {
					t.Errorf("OverParsed = %v, want 30s", got.Load.Ramp.OverParsed)
				}
			},
		},

		// ─── Steps ────────────────────────────────────────────────────────────
		{
			name: "steps empty",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps = nil; return s }(),
			wantErr: true, wantErrField: "steps",
		},
		{
			name: "step name missing",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Name = ""; return s }(),
			wantErr: true, wantErrField: "steps[0].name",
		},
		{
			name: "step duplicate name",
			spec: func() *config.ScenarioSpec {
				s := validSpec()
				s.Steps = append(s.Steps, config.StepSpec{Name: "s1", Method: "POST", Path: "/other"})
				return s
			}(),
			wantErr: true, wantErrField: "steps[1].name",
		},
		{
			name: "step method invalid",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Method = "FETCH"; return s }(),
			wantErr: true, wantErrField: "steps[0].method",
		},
		{
			name: "step method lowercase get",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Method = "get"; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Steps[0].Method != "GET" {
					t.Errorf("Method = %q, want GET", got.Steps[0].Method)
				}
			},
		},
		{
			name: "step method with whitespace",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Method = " GET "; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Steps[0].Method != "GET" {
					t.Errorf("Method = %q, want GET", got.Steps[0].Method)
				}
			},
		},
		{
			name: "step path empty",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Path = ""; return s }(),
			wantErr: true, wantErrField: "steps[0].path",
		},
		{
			name: "step path missing leading slash",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Path = "users"; return s }(),
			wantErr: true, wantErrField: "steps[0].path",
		},
		{
			name: "step body + body_raw both set",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Body = "hello"; s.Steps[0].BodyRaw = "world"; return s }(),
			wantErr: true, wantErrField: "steps[0].body",
		},
		{
			name: "step body + body_form both set",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Body = "hello"; s.Steps[0].BodyForm = map[string]string{"k": "v"}; return s }(),
			wantErr: true, wantErrField: "steps[0].body",
		},
		{
			name: "step body_raw + body_form both set",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].BodyRaw = "hello"; s.Steps[0].BodyForm = map[string]string{"k": "v"}; return s }(),
			wantErr: true, wantErrField: "steps[0].body",
		},
		{
			name: "step body + body_raw + body_form all set",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Body = "a"; s.Steps[0].BodyRaw = "b"; s.Steps[0].BodyForm = map[string]string{"c": "d"}; return s }(),
			wantErr: true, wantErrField: "steps[0].body",
		},
		{
			name: "step expect status + status_range both set",
			spec: func() *config.ScenarioSpec {
				s := validSpec()
				s.Steps[0].Expect = config.ExpectSpec{Status: 200, StatusRange: [2]int{200, 299}}
				return s
			}(),
			wantErr: true, wantErrField: "steps[0].expect",
		},
		{
			name: "step extract value with unknown prefix",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Extract = map[string]string{"myvar": "unknown:value"}; return s }(),
			wantErr: true, wantErrField: "steps[0].extract[myvar]",
		},
		{
			name: "step extract valid jsonpath",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Extract = map[string]string{"var": "$.data.id"}; return s }(),
		},
		{
			name: "step extract valid regex",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Extract = map[string]string{"var": "regex:(\\d+)"}; return s }(),
		},
		{
			name: "step extract valid header",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Extract = map[string]string{"var": "header:X-Request-Id"}; return s }(),
		},
		{
			name: "step timeout invalid",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Timeout = "not-a-duration"; return s }(),
			wantErr: true, wantErrField: "steps[0].timeout",
		},
		{
			name: "step timeout valid",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Steps[0].Timeout = "5s"; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Steps[0].TimeoutParsed != 5*time.Second {
					t.Errorf("TimeoutParsed = %v, want 5s", got.Steps[0].TimeoutParsed)
				}
			},
		},

		// ─── Options ──────────────────────────────────────────────────────────
		{
			name: "options.timeout invalid",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Options.Timeout = "bad"; return s }(),
			wantErr: true, wantErrField: "options.timeout",
		},
		{
			name: "options.timeout valid",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Options.Timeout = "10s"; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Options.TimeoutParsed != 10*time.Second {
					t.Errorf("TimeoutParsed = %v, want 10s", got.Options.TimeoutParsed)
				}
			},
		},
		{
			name: "options.max_redirects default",
			spec: validSpec(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Options.MaxRedirects != 10 {
					t.Errorf("MaxRedirects = %d, want 10 (default)", got.Options.MaxRedirects)
				}
			},
		},
		{
			name: "options.max_redirects explicitly set",
			spec: func() *config.ScenarioSpec { s := validSpec(); s.Options.MaxRedirects = 5; return s }(),
			check: func(t *testing.T, got *config.ScenarioSpec) {
				if got.Options.MaxRedirects != 5 {
					t.Errorf("MaxRedirects = %d, want 5", got.Options.MaxRedirects)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := config.Validate(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				var ce *config.ConfigError
				if !errors.As(err, &ce) {
					t.Fatalf("expected *config.ConfigError, got %T: %v", err, err)
				}
				if ce.Field != tt.wantErrField {
					t.Errorf("wantErrField = %q, got %q", tt.wantErrField, ce.Field)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, tt.spec)
			}
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	t.Parallel()
	err := &config.ConfigError{Field: "name", Message: "must be a non-empty string"}
	want := `config error: name: must be a non-empty string`
	if got := err.Error(); got != want {
		t.Errorf("ConfigError.Error() = %q, want %q", got, want)
	}
}
