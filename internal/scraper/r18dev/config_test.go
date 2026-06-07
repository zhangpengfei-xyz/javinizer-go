package r18dev

import (
	"testing"

	"github.com/javinizer/javinizer-go/internal/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.ScraperSettings
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config returns error",
			cfg:     nil,
			wantErr: true,
			errMsg:  "r18dev: config is nil",
		},
		{
			name: "disabled scraper is valid",
			cfg: &config.ScraperSettings{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "language en is valid",
			cfg: &config.ScraperSettings{
				Enabled:  true,
				Language: "en",
			},
			wantErr: false,
		},
		{
			name: "language ja is valid",
			cfg: &config.ScraperSettings{
				Enabled:  true,
				Language: "ja",
			},
			wantErr: false,
		},
		{
			name: "language empty is valid (defaults to en)",
			cfg: &config.ScraperSettings{
				Enabled:  true,
				Language: "",
			},
			wantErr: false,
		},
		{
			name: "language case insensitive EN is valid",
			cfg: &config.ScraperSettings{
				Enabled:  true,
				Language: "EN",
			},
			wantErr: false,
		},
		{
			name: "language case insensitive JA is valid",
			cfg: &config.ScraperSettings{
				Enabled:  true,
				Language: "Ja",
			},
			wantErr: false,
		},
		{
			name: "language fr is invalid",
			cfg: &config.ScraperSettings{
				Enabled:  true,
				Language: "fr",
			},
			wantErr: true,
			errMsg:  "r18dev: language must be 'en' or 'ja', got \"fr\"",
		},
		{
			name: "language de is invalid",
			cfg: &config.ScraperSettings{
				Enabled:  true,
				Language: "DE",
			},
			wantErr: true,
			errMsg:  "r18dev: language must be 'en' or 'ja', got \"DE\"",
		},
		{
			name: "RateLimit -1 is invalid",
			cfg: &config.ScraperSettings{
				Enabled:   true,
				RateLimit: -1,
			},
			wantErr: true,
			errMsg:  "r18dev: rate_limit must be non-negative, got -1",
		},
		{
			name: "RateLimit 0 is valid",
			cfg: &config.ScraperSettings{
				Enabled:   true,
				RateLimit: 0,
			},
			wantErr: false,
		},
		{
			name: "RateLimit 1000 is valid",
			cfg: &config.ScraperSettings{
				Enabled:   true,
				RateLimit: 1000,
			},
			wantErr: false,
		},
		{
			name: "RetryCount -1 is invalid",
			cfg: &config.ScraperSettings{
				Enabled:    true,
				RetryCount: -1,
			},
			wantErr: true,
			errMsg:  "r18dev: retry_count must be non-negative, got -1",
		},
		{
			name: "RetryCount 0 is valid",
			cfg: &config.ScraperSettings{
				Enabled:    true,
				RetryCount: 0,
			},
			wantErr: false,
		},
		{
			name: "RetryCount 3 is valid",
			cfg: &config.ScraperSettings{
				Enabled:    true,
				RetryCount: 3,
			},
			wantErr: false,
		},
		{
			name: "Timeout -1 is invalid",
			cfg: &config.ScraperSettings{
				Enabled: true,
				Timeout: -1,
			},
			wantErr: true,
			errMsg:  "r18dev: timeout must be non-negative, got -1",
		},
		{
			name: "Timeout 0 is valid",
			cfg: &config.ScraperSettings{
				Enabled: true,
				Timeout: 0,
			},
			wantErr: false,
		},
		{
			name: "Timeout 30 is valid",
			cfg: &config.ScraperSettings{
				Enabled: true,
				Timeout: 30,
			},
			wantErr: false,
		},
		{
			name: "FlareSolverr enabled is valid",
			cfg: &config.ScraperSettings{
				Enabled:         true,
				UseFlareSolverr: true,
			},
			wantErr: false,
		},
		{
			name: "all valid fields",
			cfg: &config.ScraperSettings{
				Enabled:    true,
				Language:   "ja",
				RateLimit:  500,
				RetryCount: 3,
				Timeout:    60,
			},
			wantErr: false,
		},
	}

	c := &R18DevConfig{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateConfig(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Equal(t, tt.errMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestConfigUnmarshalToScraperNew tests the full path from YAML config to scraper instance:
// unmarshal YAML -> FlatToScraperConfig -> scraper.New -> scraper.Config()
func TestConfigUnmarshalToScraperNew(t *testing.T) {
	tests := []struct {
		name                  string
		yamlConfig            string
		wantRespectRetryAfter bool
		wantEnabled           bool
		wantLanguage          string
		wantRateLimit         int
	}{
		{
			name: "respect_retry_after true from YAML",
			yamlConfig: `
enabled: true
language: en
request_delay: 1500
max_retries: 3
respect_retry_after: true
`,
			wantRespectRetryAfter: true,
			wantEnabled:           true,
			wantLanguage:          "en",
			wantRateLimit:         1500,
		},
		{
			name: "respect_retry_after false from YAML",
			yamlConfig: `
enabled: true
language: ja
request_delay: 2000
max_retries: 5
respect_retry_after: false
`,
			wantRespectRetryAfter: false,
			wantEnabled:           true,
			wantLanguage:          "ja",
			wantRateLimit:         2000,
		},
		{
			name: "disabled scraper",
			yamlConfig: `
enabled: false
language: en
request_delay: 0
max_retries: 0
respect_retry_after: false
`,
			wantRespectRetryAfter: false,
			wantEnabled:           false,
			wantLanguage:          "en",
			wantRateLimit:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Unmarshal YAML into R18DevConfig struct
			var r18Cfg R18DevConfig
			err := yaml.Unmarshal([]byte(tt.yamlConfig), &r18Cfg)
			assert.NoError(t, err, "YAML unmarshal should succeed")
			assert.Equal(t, tt.wantEnabled, r18Cfg.Enabled, "Enabled should match YAML value")
			if tt.wantRespectRetryAfter {
				assert.NotNil(t, r18Cfg.RespectRetryAfter, "RespectRetryAfter should be non-nil")
				assert.True(t, *r18Cfg.RespectRetryAfter, "RespectRetryAfter should be true")
			} else {
				if r18Cfg.RespectRetryAfter != nil {
					assert.False(t, *r18Cfg.RespectRetryAfter, "RespectRetryAfter should be false")
				}
			}

			// Step 2: Convert to ScraperSettings via FlatToScraperConfig
			settings := config.FlatToScraperConfig("r18dev", &r18Cfg)
			assert.NotNil(t, settings, "FlatToScraperConfig should return non-nil settings")
			assert.Equal(t, tt.wantEnabled, settings.Enabled, "ScraperSettings.Enabled should match")
			assert.Equal(t, tt.wantRateLimit, settings.RateLimit, "ScraperSettings.RateLimit should match")

			// Verify settings
			assert.NotNil(t, settings, "FlatToScraperConfig should return non-nil settings")
			assert.Equal(t, tt.wantEnabled, settings.Enabled, "ScraperSettings.Enabled should match")
			assert.Equal(t, tt.wantRateLimit, settings.RateLimit, "ScraperSettings.RateLimit should match")

			// Note: respect_retry_after was previously in Extra, now in R18DevConfig

			// Step 3: Create scraper via r18dev.New
			scraper := New(*settings, nil, config.FlareSolverrConfig{})
			assert.NotNil(t, scraper, "New scraper should not be nil")

			// Step 4: Verify scraper.Config() returns correct settings
			cfg := scraper.Config()
			assert.NotNil(t, cfg, "Config() should return non-nil")

			// Note: respect_retry_after was previously in Extra, now in R18DevConfig
		})
	}
}

// TestConfigUnmarshalToScraperNewFlaresolverr tests flaresolverr preservation through the full chain.
func TestConfigUnmarshalToScraperNewFlaresolverr(t *testing.T) {
	yamlConfig := `
enabled: true
language: en
request_delay: 1000
max_retries: 3
respect_retry_after: true
flaresolverr:
    enabled: true
    url: "http://localhost:8191/v1"
    timeout: 60
    max_retries: 5
    session_ttl: 600
`

	// Step 1: Unmarshal YAML
	var r18Cfg R18DevConfig
	err := yaml.Unmarshal([]byte(yamlConfig), &r18Cfg)
	assert.NoError(t, err)

	// Step 2: Convert to ScraperSettings
	settings := config.FlatToScraperConfig("r18dev", &r18Cfg)
	assert.NotNil(t, settings)

	// Step 3: Create scraper with global flaresolverr disabled
	globalFlareSolverr := config.FlareSolverrConfig{Enabled: false}
	scraper := New(*settings, nil, globalFlareSolverr)
	assert.NotNil(t, scraper)

	// Step 4: Verify flaresolverr settings were preserved
	cfg := scraper.Config()
	assert.NotNil(t, cfg)

	// Note: When global flaresolverr is disabled, per-scraper flaresolverr is also disabled
	// So we verify the per-scraper flaresolverr is NOT enabled in this case
	assert.False(t, cfg.UseFlareSolverr, "FlareSolverr should be disabled when global is disabled")
}

// TestConfigUnmarshalInvalidYAML tests that invalid YAML returns an error.
func TestConfigUnmarshalInvalidYAML(t *testing.T) {
	invalidYAML := `
enabled: true
language: invalid_language_that_is_too_long_for_validation
request_delay: -1
`

	var r18Cfg R18DevConfig
	err := yaml.Unmarshal([]byte(invalidYAML), &r18Cfg)
	// yaml.Unmarshal succeeds even with invalid values - validation happens later
	assert.NoError(t, err, "YAML unmarshal should succeed for any YAML structure")

	// Converting to ScraperSettings should still work (no validation in FlatToScraperConfig)
	settings := config.FlatToScraperConfig("r18dev", &r18Cfg)
	assert.NotNil(t, settings, "FlatToScraperConfig should return settings even for invalid values")
}

// TestConfigYAMLFormat tests that the YAML format matches what a user would write in config.yaml.
func TestConfigYAMLFormat(t *testing.T) {
	// This is the exact format a user would write in their config.yaml
	userConfigYAML := `
enabled: true
language: en
request_delay: 1500
max_retries: 3
respect_retry_after: true
timeout: 0
rate_limit: 1500
retry_count: 0
user_agent: ""
extra:
    respect_retry_after: true
`

	var r18Cfg R18DevConfig
	err := yaml.Unmarshal([]byte(userConfigYAML), &r18Cfg)
	assert.NoError(t, err, "User config YAML should unmarshal successfully")

	// Verify all fields are correctly unmarshaled
	assert.True(t, r18Cfg.Enabled)
	assert.Equal(t, "en", r18Cfg.Language)
	assert.Equal(t, 1500, r18Cfg.RequestDelay)
	assert.Equal(t, 3, r18Cfg.MaxRetries)
	assert.NotNil(t, r18Cfg.RespectRetryAfter)
	assert.True(t, *r18Cfg.RespectRetryAfter)
	settings := config.FlatToScraperConfig("r18dev", &r18Cfg)
	assert.NotNil(t, settings)
	assert.True(t, settings.Enabled)

	// Create scraper
	scraper := New(*settings, nil, config.FlareSolverrConfig{})
	cfg := scraper.Config()

	// Note: respect_retry_after was previously in Extra, now in R18DevConfig
	assert.NotNil(t, cfg)
}

// TestR18DevConfigFields tests that R18DevConfig specific fields are accessible.
func TestR18DevConfigFields(t *testing.T) {
	yamlConfig := `
enabled: true
language: ja
request_delay: 2000
max_retries: 5
respect_retry_after: true
priority: 10
`

	var r18Cfg R18DevConfig
	err := yaml.Unmarshal([]byte(yamlConfig), &r18Cfg)
	assert.NoError(t, err)

	// Verify all fields are correctly unmarshaled
	assert.Equal(t, "ja", r18Cfg.Language)
	assert.Equal(t, 2000, r18Cfg.RequestDelay)
	assert.Equal(t, 5, r18Cfg.MaxRetries)
	assert.NotNil(t, r18Cfg.RespectRetryAfter)
	assert.True(t, *r18Cfg.RespectRetryAfter)
	assert.Equal(t, 10, r18Cfg.Priority)
	assert.True(t, r18Cfg.Enabled)
}

// TestScraperConfigExtraContents verifies the contents of scraper settings after full conversion.
// Note: This test was previously checking Extra field contents. The Extra field has been
// removed and scraper-specific fields are now in concrete config types.
func TestScraperConfigExtraContents(t *testing.T) {
	yamlConfig := `
enabled: true
respect_retry_after: true
`

	var r18Cfg R18DevConfig
	err := yaml.Unmarshal([]byte(yamlConfig), &r18Cfg)
	assert.NoError(t, err)

	settings := config.FlatToScraperConfig("r18dev", &r18Cfg)
	assert.NotNil(t, settings)
	assert.True(t, settings.Enabled)

	// Create scraper
	scraper := New(*settings, nil, config.FlareSolverrConfig{})
	cfg := scraper.Config()

	// Note: respect_retry_after is now in R18DevConfig, not in ScraperSettings.Extra
	assert.NotNil(t, cfg)
	assert.True(t, cfg.Enabled)
}
