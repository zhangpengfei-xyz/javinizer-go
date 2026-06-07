package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/javinizer/javinizer-go/internal/scraperutil"
	"gopkg.in/yaml.v3"
)

// ConfigValidator is implemented by flat per-scraper config structs.
// Each scraper config validates its scraper-specific fields via ValidateConfig().
// CONF-03, CONF-04: Enables interface-based dispatch without hardcoded scraper-name branches.
type ConfigValidator interface {
	ValidateConfig(*ScraperSettings) error
}

// ScraperSettingsAdapter is an optional adapter for concrete scraper configs.
// When implemented, config decoding can avoid flatten-registry coupling.
type ScraperSettingsAdapter interface {
	ToScraperSettings() *ScraperSettings
}

// ScrapersConfig holds scraper-specific settings.
// PLUGIN-01: No concrete scraper type fields - map-backed storage only.
type ScrapersConfig struct {
	UserAgent             string             `yaml:"user_agent" json:"user_agent"`
	Referer               string             `yaml:"referer" json:"referer"`                                 // Referer header for CDN compatibility (default: https://www.dmm.co.jp/)
	TimeoutSeconds        int                `yaml:"timeout_seconds" json:"timeout_seconds"`                 // HTTP client timeout in seconds (default: 30)
	RequestTimeoutSeconds int                `yaml:"request_timeout_seconds" json:"request_timeout_seconds"` // Overall request timeout in seconds (default: 60)
	Priority              []string           `yaml:"priority" json:"priority"`                               // Global scraper priority order
	FlareSolverr          FlareSolverrConfig `yaml:"flaresolverr" json:"flaresolverr"`                       // Global FlareSolverr config for Cloudflare bypass
	// NEW: Global scrape_actress default (opt-out behavior, default: true)
	ScrapeActress bool `yaml:"scrape_actress" json:"scrape_actress"`

	// NEW: Global Browser configuration block
	Browser     BrowserConfig               `yaml:"browser" json:"browser"`
	Proxy       ProxyConfig                 `yaml:"proxy" json:"proxy"` // Default HTTP/SOCKS5 proxy for scraper requests
	Overrides   map[string]*ScraperSettings `yaml:"-" json:"-"`         // Canonical per-scraper settings map
	flatConfigs map[string]ConfigValidator  `yaml:"-" json:"-"`         // Validator dispatch table built by NormalizeScraperConfigs
}

type scraperDecodeFormat int

const (
	scraperDecodeYAML scraperDecodeFormat = iota
	scraperDecodeJSON
)

// UnmarshalYAML implements custom YAML unmarshaling for ScrapersConfig.
func (s *ScrapersConfig) UnmarshalYAML(node *yaml.Node) error {
	if node == nil || node.Kind == 0 {
		s.Overrides = make(map[string]*ScraperSettings)
		return nil
	}

	var generic map[string]any
	if err := node.Decode(&generic); err != nil {
		return fmt.Errorf("failed to unmarshal scrapers config: %w", err)
	}

	return s.decodeFromGeneric(generic, scraperDecodeYAML)
}

// UnmarshalJSON implements custom JSON unmarshaling for ScrapersConfig.
func (s *ScrapersConfig) UnmarshalJSON(data []byte) error {
	var generic map[string]any
	if err := json.Unmarshal(data, &generic); err != nil {
		return fmt.Errorf("failed to unmarshal scrapers config: %w", err)
	}
	return s.decodeFromGeneric(generic, scraperDecodeJSON)
}

func (s *ScrapersConfig) decodeFromGeneric(generic map[string]any, format scraperDecodeFormat) error {
	// Always reset map state on unmarshal to avoid stale entries when reusing structs.
	s.Overrides = make(map[string]*ScraperSettings)

	if generic == nil {
		return nil
	}

	for key, value := range generic {
		switch key {
		case "user_agent":
			v, ok := value.(string)
			if !ok {
				return fmt.Errorf("user_agent must be a string")
			}
			s.UserAgent = v
		case "referer":
			v, ok := value.(string)
			if !ok {
				return fmt.Errorf("referer must be a string")
			}
			s.Referer = v
		case "timeout_seconds":
			v, ok := toInt(value)
			if !ok {
				return fmt.Errorf("timeout_seconds must be an integer")
			}
			s.TimeoutSeconds = v
		case "request_timeout_seconds":
			v, ok := toInt(value)
			if !ok {
				return fmt.Errorf("request_timeout_seconds must be an integer")
			}
			s.RequestTimeoutSeconds = v
		case "priority":
			if value == nil {
				s.Priority = nil
				continue
			}
			values, ok := value.([]any)
			if !ok {
				return fmt.Errorf("priority must be an array of strings")
			}
			s.Priority = make([]string, 0, len(values))
			for i, elem := range values {
				str, ok := elem.(string)
				if !ok {
					return fmt.Errorf("priority[%d] must be a string", i)
				}
				s.Priority = append(s.Priority, str)
			}
		case "proxy":
			data, err := marshalByFormat(format, value)
			if err != nil {
				return fmt.Errorf("failed to marshal proxy: %w", err)
			}
			if err := unmarshalByFormat(format, data, &s.Proxy); err != nil {
				return fmt.Errorf("failed to unmarshal proxy: %w", err)
			}
		case "flaresolverr":
			data, err := marshalByFormat(format, value)
			if err != nil {
				return fmt.Errorf("failed to marshal flaresolverr: %w", err)
			}
			if err := unmarshalByFormat(format, data, &s.FlareSolverr); err != nil {
				return fmt.Errorf("failed to unmarshal flaresolverr: %w", err)
			}

		// NEW: Handle scrape_actress
		case "scrape_actress":
			v, ok := value.(bool)
			if !ok {
				return fmt.Errorf("scrape_actress must be a boolean")
			}
			s.ScrapeActress = v

		// NEW: Handle browser
		case "browser":
			data, err := marshalByFormat(format, value)
			if err != nil {
				return fmt.Errorf("failed to marshal browser: %w", err)
			}
			if err := unmarshalByFormat(format, data, &s.Browser); err != nil {
				return fmt.Errorf("failed to unmarshal browser: %w", err)
			}

		default:
			ss, err := decodeScraperEntry(key, value, format)
			if err != nil {
				return fmt.Errorf("failed to parse config for scraper %q: %w", key, err)
			}
			s.Overrides[key] = ss
		}
	}

	return nil
}

func decodeScraperEntry(name string, value any, format scraperDecodeFormat) (*ScraperSettings, error) {
	factory := scraperutil.GetConfigFactory(name)
	if factory == nil && !isSupportedScraperName(name) {
		return nil, fmt.Errorf("unknown scraper %q", name)
	}

	data, err := marshalByFormat(format, value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config for scraper %q: %w", name, err)
	}

	var raw map[string]any
	if err := unmarshalByFormat(format, data, &raw); err != nil {
		return nil, fmt.Errorf("failed to decode generic config for scraper %q: %w", name, err)
	}

	// Scrapers with a registered concrete config factory are validated strictly for
	// known fields while allowing unified ScraperSettings keys for compatibility.
	// Supported scraper names without a registered factory decode generically.
	if factory != nil {
		concrete := factory()
		knownKeys := scraperConfigKeys(concrete)
		for key := range raw {
			if _, ok := knownKeys[key]; ok {
				continue
			}
			if isAllowedUnifiedScraperKey(key) {
				if err := validateUnifiedScraperField(key, raw[key], format); err != nil {
					return nil, fmt.Errorf("invalid field %q: %w", key, err)
				}
				continue
			}
			return nil, fmt.Errorf("unknown field %q", key)
		}

		knownOnly := make(map[string]any, len(raw))
		for key, value := range raw {
			if _, ok := knownKeys[key]; ok {
				knownOnly[key] = value
			}
		}

		knownData, err := marshalByFormat(format, knownOnly)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal known fields for scraper %q: %w", name, err)
		}
		if err := strictDecodeConcrete(format, knownData, concrete); err != nil {
			return nil, err
		}
	}

	ss, err := decodeGenericScraperSettings(raw, func(dst any) error {
		return unmarshalByFormat(format, data, dst)
	})
	if err != nil {
		return nil, err
	}
	return &ss, nil
}

func isSupportedScraperName(name string) bool {
	if name == "" {
		return false
	}

	_, exists := scraperutil.GetDefaultScraperSettings()[name]
	return exists
}

func scraperConfigKeys(concrete any) map[string]struct{} {
	keys := make(map[string]struct{})
	if concrete == nil {
		return keys
	}

	typ := reflect.TypeOf(concrete)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return keys
	}

	collectStructKeys(typ, keys)
	return keys
}

func collectStructKeys(typ reflect.Type, keys map[string]struct{}) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}

		yamlTag := strings.TrimSpace(field.Tag.Get("yaml"))
		if yamlTag == ",inline" {
			embedType := field.Type
			if embedType.Kind() == reflect.Ptr {
				embedType = embedType.Elem()
			}
			if embedType.Kind() == reflect.Struct {
				collectStructKeys(embedType, keys)
			}
			continue
		}

		yamlTagged := yamlTag != ""
		if yamlTag != "" {
			name := strings.Split(yamlTag, ",")[0]
			if name != "" && name != "-" {
				keys[name] = struct{}{}
			}
		}

		jsonTag := strings.TrimSpace(field.Tag.Get("json"))
		jsonTagged := jsonTag != ""
		if jsonTag != "" {
			name := strings.Split(jsonTag, ",")[0]
			if name != "" && name != "-" {
				keys[name] = struct{}{}
			}
		}

		if !yamlTagged && !jsonTagged {
			keys[field.Name] = struct{}{}
			keys[strings.ToLower(field.Name)] = struct{}{}
		}
	}
}

func isAllowedUnifiedScraperKey(key string) bool {
	switch key {
	case "enabled", "language", "timeout", "rate_limit", "retry_count", "user_agent",
		"proxy", "download_proxy", "use_flaresolverr", "base_url", "cookies",
		"request_delay", "max_retries", "respect_retry_after",
		// NEW: Allow use_browser and scrape_actress per-scraper overrides
		"use_browser", "scrape_actress":
		return true
	default:
		// Allow any key starting with known scraper-specific prefixes
		// This permits extra fields to be stored in Extra map
		return false // Unknown keys go to Extra
	}
}

func validateUnifiedScraperField(key string, value any, format scraperDecodeFormat) error {
	data, err := marshalByFormat(format, value)
	if err != nil {
		return err
	}

	switch key {
	case "enabled":
		var v bool
		return unmarshalByFormat(format, data, &v)
	case "language", "user_agent":
		var v string
		return unmarshalByFormat(format, data, &v)
	case "timeout", "rate_limit", "retry_count", "request_delay", "max_retries":
		var v int
		return unmarshalByFormat(format, data, &v)
	case "proxy", "download_proxy":
		var v *ProxyConfig
		return strictDecodeConcrete(format, data, &v)
	case "use_flaresolverr":
		var v bool
		return unmarshalByFormat(format, data, &v)
	case "use_browser":
		var v bool
		return unmarshalByFormat(format, data, &v)
	case "scrape_actress":
		var v bool
		return unmarshalByFormat(format, data, &v)
	case "respect_retry_after":
		var v bool
		return unmarshalByFormat(format, data, &v)
	case "cookies":
		var v map[string]string
		return unmarshalByFormat(format, data, &v)
	default:
		return fmt.Errorf("unsupported unified field %q", key)
	}
}

func strictDecodeConcrete(format scraperDecodeFormat, data []byte, dst any) error {
	switch format {
	case scraperDecodeYAML:
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		return decoder.Decode(dst)
	case scraperDecodeJSON:
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		return decoder.Decode(dst)
	default:
		return fmt.Errorf("unknown decode format: %d", format)
	}
}

func marshalByFormat(format scraperDecodeFormat, value any) ([]byte, error) {
	switch format {
	case scraperDecodeYAML:
		return yaml.Marshal(value)
	case scraperDecodeJSON:
		return json.Marshal(value)
	default:
		return nil, fmt.Errorf("unknown decode format: %d", format)
	}
}

func unmarshalByFormat(format scraperDecodeFormat, data []byte, dst any) error {
	switch format {
	case scraperDecodeYAML:
		return yaml.Unmarshal(data, dst)
	case scraperDecodeJSON:
		return json.Unmarshal(data, dst)
	default:
		return fmt.Errorf("unknown decode format: %d", format)
	}
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		fv := float64(v)
		if math.Trunc(fv) != fv {
			return 0, false
		}
		return int(fv), true
	case float64:
		if math.Trunc(v) != v {
			return 0, false
		}
		return int(v), true
	default:
		return 0, false
	}
}

func decodeGenericScraperSettings(raw map[string]any, decode func(any) error) (ScraperSettings, error) {
	var ss ScraperSettings
	if err := decode(&ss); err != nil {
		return ScraperSettings{}, err
	}

	// Compatibility aliases used by concrete scraper configs.
	if ss.RateLimit == 0 {
		if requestDelay, ok := toInt(raw["request_delay"]); ok {
			ss.RateLimit = requestDelay
		}
	}
	if ss.RetryCount == 0 {
		if maxRetries, ok := toInt(raw["max_retries"]); ok {
			ss.RetryCount = maxRetries
		}
	}

	// Preserve unknown keys in Extra for scraper-specific fields
	extra := make(map[string]any)
	for key, value := range raw {
		if !isAllowedUnifiedScraperKey(key) {
			extra[key] = value
		}
	}
	if len(extra) > 0 {
		ss.Extra = extra
	}

	return ss, nil
}

// MarshalJSON implements custom JSON marshaling for ScrapersConfig.
// This ensures scraper-specific settings are serialized despite json:"-" on internal maps.
func (s *ScrapersConfig) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)

	m["user_agent"] = s.UserAgent
	m["referer"] = s.Referer
	m["timeout_seconds"] = s.TimeoutSeconds
	m["request_timeout_seconds"] = s.RequestTimeoutSeconds
	m["priority"] = s.Priority
	m["proxy"] = s.Proxy
	m["flaresolverr"] = s.FlareSolverr

	// NEW: Include scrape_actress and browser
	m["scrape_actress"] = s.ScrapeActress
	m["browser"] = s.Browser

	for name, settings := range s.Overrides {
		if settings != nil {
			m[name] = settings
		}
	}

	return json.Marshal(m)
}

// MarshalYAML serializes scrapers with full unified ScraperSettings.
func (s *ScrapersConfig) MarshalYAML() (interface{}, error) {
	m := make(map[string]any)

	m["user_agent"] = s.UserAgent
	m["referer"] = s.Referer
	m["timeout_seconds"] = s.TimeoutSeconds
	m["request_timeout_seconds"] = s.RequestTimeoutSeconds
	m["priority"] = s.Priority
	m["proxy"] = s.Proxy
	m["flaresolverr"] = s.FlareSolverr

	// NEW: Include scrape_actress and browser
	m["scrape_actress"] = s.ScrapeActress
	m["browser"] = s.Browser

	for name, settings := range s.Overrides {
		if settings != nil {
			m[name] = settings
		}
	}

	return m, nil
}

// delegatingValidator wraps a scraperutil.ValidatorFunc to implement ConfigValidator.
type delegatingValidator struct {
	name string
	fn   scraperutil.ValidatorFunc
}

func (v *delegatingValidator) ValidateConfig(sc *ScraperSettings) error {
	return v.fn(sc)
}

func defaultScraperSettingsCopy(value any) *ScraperSettings {
	switch v := value.(type) {
	case ScraperSettings:
		return v.DeepCopy()
	case *ScraperSettings:
		if v == nil {
			return nil
		}
		return v.DeepCopy()
	default:
		return nil
	}
}

// NormalizeScraperConfigs populates Overrides and flatConfigs from registered defaults
// and validators. Overrides is the canonical per-scraper configuration map.
func (c *ScrapersConfig) NormalizeScraperConfigs() {
	if c.Overrides == nil {
		c.Overrides = make(map[string]*ScraperSettings)
	}
	if c.flatConfigs == nil {
		c.flatConfigs = make(map[string]ConfigValidator)
	}

	// Always rebuild validator dispatch from current registry to reflect test/runtime changes.
	for name := range c.flatConfigs {
		delete(c.flatConfigs, name)
	}

	registeredDefaults := scraperutil.GetDefaultScraperSettings()
	for name, defaultSettings := range registeredDefaults {
		if c.Overrides[name] == nil {
			if copied := defaultScraperSettingsCopy(defaultSettings); copied != nil {
				c.Overrides[name] = copied
			}
		}
	}

	for name := range c.Overrides {
		if validatorFn := scraperutil.GetValidator(name); validatorFn != nil {
			c.flatConfigs[name] = &delegatingValidator{name: name, fn: validatorFn}
		}
	}
}

// FlatToScraperConfig converts a flat per-scraper config to unified ScraperSettings.
// Uses FlattenFunc registry for backwards compatibility with existing scraper packages.
func FlatToScraperConfig(name string, flat interface{}) *ScraperSettings {
	if flat == nil {
		return nil
	}

	if adapter, ok := flat.(ScraperSettingsAdapter); ok {
		return adapter.ToScraperSettings()
	}

	fn := scraperutil.GetFlattenFunc(name)
	if fn == nil {
		return nil
	}

	iface, ok := flat.(scraperutil.ScraperConfigInterface)
	if !ok {
		return nil
	}

	result := fn(iface)
	if result == nil {
		return nil
	}

	return result.(*ScraperSettings)
}
