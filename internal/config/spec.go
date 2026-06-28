package config

import "time"

// ScenarioSpec is the top-level struct. Maps 1:1 to the YAML file.
type ScenarioSpec struct {
    Name       string      `yaml:"name"`
    BaseURL    string      `yaml:"base_url"`
    Load       LoadSpec    `yaml:"load"`
    Options    OptionsSpec `yaml:"options"`
    Steps      []StepSpec  `yaml:"steps"`
    Thresholds ThresholdSpec `yaml:"thresholds"`
    Regression RegressionSpec `yaml:"regression"`
}

type LoadSpec struct {
    Duration string   `yaml:"duration"`        // e.g. "60s", "5m"
    Users    int      `yaml:"users"`
    Rate     string   `yaml:"rate"`            // e.g. "100/s", "50/m"
    Ramp     *RampSpec `yaml:"ramp,omitempty"`

    // Parsed forms: populated by validator, not from YAML directly
    DurationParsed time.Duration `yaml:"-"`
    RatePerSecond  float64       `yaml:"-"`
}

type RampSpec struct {
    From string `yaml:"from"` // e.g. "10/s"
    To   string `yaml:"to"`   // e.g. "100/s"
    Over string `yaml:"over"` // e.g. "30s"

    // Parsed forms
    FromRPS      float64       `yaml:"-"`
    ToRPS        float64       `yaml:"-"`
    OverParsed   time.Duration `yaml:"-"`
}

type OptionsSpec struct {
    CookieJar        bool              `yaml:"cookie_jar"`
    FollowRedirects  bool              `yaml:"follow_redirects"`
    MaxRedirects     int               `yaml:"max_redirects"`
    Timeout          string            `yaml:"timeout"`
    HTTP2            bool              `yaml:"http2"`
    DNSOverride      map[string]string `yaml:"dns_override"`

    // Parsed forms
    TimeoutParsed time.Duration `yaml:"-"`
}

type StepSpec struct {
    Name    string            `yaml:"name"`
    Method  string            `yaml:"method"`
    Path    string            `yaml:"path"`
    Headers map[string]string `yaml:"headers"`
    Body    any               `yaml:"body"`       // map or string
    BodyRaw string            `yaml:"body_raw"`
    BodyForm map[string]string `yaml:"body_form"`
    Timeout string            `yaml:"timeout"`
    Expect  ExpectSpec        `yaml:"expect"`
    Extract map[string]string `yaml:"extract"`    // varName → "$.json.path" or "regex:..." or "header:Name"

    TimeoutParsed time.Duration `yaml:"-"`

    // Populated by template engine during init
    IsStaticPath    bool `yaml:"-"`
    IsStaticBody    bool `yaml:"-"`
    IsStaticHeaders bool `yaml:"-"`
}

type ExpectSpec struct {
    Status       int               `yaml:"status"`
    StatusRange  [2]int            `yaml:"status_range"`
    LatencyMs    int               `yaml:"latency_ms"`
    BodyContains string            `yaml:"body_contains"`
    BodyMatches  string            `yaml:"body_matches"`
    HeaderPresent string           `yaml:"header_present"`
    HeaderEquals  map[string]string `yaml:"header_equals"`
}

type ThresholdSpec struct {
    P99Ms     float64              `yaml:"p99_ms"`
    P95Ms     float64              `yaml:"p95_ms"`
    P50Ms     float64              `yaml:"p50_ms"`
    ErrorRate float64              `yaml:"error_rate"`
    Steps     []StepThresholdSpec  `yaml:"steps"`
}

type StepThresholdSpec struct {
    Name      string  `yaml:"name"`
    P99Ms     float64 `yaml:"p99_ms"`
    P95Ms     float64 `yaml:"p95_ms"`
    ErrorRate float64 `yaml:"error_rate"`
}

type RegressionSpec struct {
    P99Percent       float64 `yaml:"p99_percent"`
    P95Percent       float64 `yaml:"p95_percent"`
    ErrorRatePercent float64 `yaml:"error_rate_percent"`
}