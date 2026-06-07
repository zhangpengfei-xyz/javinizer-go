package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScraperSettings_GetBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings ScraperSettings
		wantURL  string
	}{
		{
			name:    "empty_base_url",
			wantURL: "",
		},
		{
			name: "with_base_url",
			settings: ScraperSettings{
				BaseURL: "https://example.com",
			},
			wantURL: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.settings.GetBaseURL()
			assert.Equal(t, tt.wantURL, got)
		})
	}
}

func TestScraperSettings_ToScraperSettings(t *testing.T) {
	t.Parallel()

	settings := &ScraperSettings{
		Enabled:    true,
		Timeout:    30,
		UserAgent:  "test-agent",
		BaseURL:    "https://example.com",
		RateLimit:  1000,
		RetryCount: 3,
	}

	result := settings.ToScraperSettings()
	assert.NotNil(t, result)
	assert.Equal(t, settings, result)
}

func TestScraperCommonConfig_Getters(t *testing.T) {
	t.Parallel()

	proxy := &ProxyConfig{Enabled: true}
	downloadProxy := &ProxyConfig{Enabled: true}

	tests := []struct {
		name              string
		config            ScraperCommonConfig
		wantEnabled       bool
		wantUserAgent     string
		wantRequestDelay  int
		wantMaxRetries    int
		wantProxy         any
		wantDownloadProxy any
	}{
		{
			name:              "all_defaults",
			config:            ScraperCommonConfig{},
			wantEnabled:       false,
			wantUserAgent:     "",
			wantRequestDelay:  0,
			wantMaxRetries:    0,
			wantProxy:         nil,
			wantDownloadProxy: nil,
		},
		{
			name: "all_set",
			config: ScraperCommonConfig{
				Enabled:       true,
				UserAgent:     UserAgentString{Value: "custom-agent"},
				RequestDelay:  500,
				MaxRetries:    3,
				Proxy:         proxy,
				DownloadProxy: downloadProxy,
			},
			wantEnabled:       true,
			wantUserAgent:     "custom-agent",
			wantRequestDelay:  500,
			wantMaxRetries:    3,
			wantProxy:         proxy,
			wantDownloadProxy: downloadProxy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantEnabled, tt.config.IsEnabled())
			assert.Equal(t, tt.wantUserAgent, tt.config.GetUserAgent())
			assert.Equal(t, tt.wantRequestDelay, tt.config.GetRequestDelay())
			assert.Equal(t, tt.wantMaxRetries, tt.config.GetMaxRetries())

			proxy := tt.config.GetProxy()
			if tt.wantProxy == nil {
				assert.Nil(t, proxy)
			} else {
				assert.Equal(t, tt.wantProxy, proxy)
			}

			dlProxy := tt.config.GetDownloadProxy()
			if tt.wantDownloadProxy == nil {
				assert.Nil(t, dlProxy)
			} else {
				assert.Equal(t, tt.wantDownloadProxy, dlProxy)
			}
		})
	}
}

func TestScraperSettings_MarshalYAML(t *testing.T) {
	t.Parallel()

	t.Run("nil_receiver", func(t *testing.T) {
		var s *ScraperSettings
		result, err := s.MarshalYAML()
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("with_extra_fields", func(t *testing.T) {
		scrapeActress := true
		s := &ScraperSettings{
			Enabled:         true,
			Language:        "en",
			Timeout:         30,
			RateLimit:       1000,
			RetryCount:      3,
			UserAgent:       "test-agent",
			BaseURL:         "https://example.com",
			UseFlareSolverr: true,
			UseBrowser:      true,
			ScrapeActress:   &scrapeActress,
			Cookies:         map[string]string{"session": "abc123"},
			Extra:           map[string]any{"custom_field": "custom_value"},
		}

		result, err := s.MarshalYAML()
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap := result.(map[string]any)
		assert.Equal(t, true, resultMap["enabled"])
		assert.Equal(t, "en", resultMap["language"])
		assert.Equal(t, 30, resultMap["timeout"])
		assert.Equal(t, "custom_value", resultMap["custom_field"])
		assert.Equal(t, true, resultMap["use_browser"])
		assert.Equal(t, true, resultMap["scrape_actress"])
	})
}

func TestScraperSettings_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		s := &ScraperSettings{
			Enabled:   true,
			Language:  "en",
			Timeout:   30,
			UserAgent: "test-agent",
		}

		data, err := s.MarshalJSON()
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"enabled":true`)
		assert.Contains(t, string(data), `"language":"en"`)
	})
}

func TestScraperSettings_SetScrapeActress(t *testing.T) {
	t.Parallel()

	t.Run("normal_case", func(t *testing.T) {
		s := &ScraperSettings{}

		s.SetScrapeActress(true)
		assert.NotNil(t, s.ScrapeActress)
		assert.True(t, *s.ScrapeActress)

		s.SetScrapeActress(false)
		assert.NotNil(t, s.ScrapeActress)
		assert.False(t, *s.ScrapeActress)
	})

	t.Run("nil_receiver", func(t *testing.T) {
		var s *ScraperSettings
		assert.NotPanics(t, func() {
			s.SetScrapeActress(true)
		})
	})
}

func TestScraperSettings_ShouldUseBrowser(t *testing.T) {
	t.Parallel()

	t.Run("nil_receiver", func(t *testing.T) {
		var s *ScraperSettings
		result := s.ShouldUseBrowser(true)
		assert.False(t, result)
	})

	t.Run("global_disabled", func(t *testing.T) {
		s := &ScraperSettings{UseBrowser: true}
		result := s.ShouldUseBrowser(false)
		assert.False(t, result, "Should return false when global is disabled")
	})

	t.Run("global_enabled_use_browser_true", func(t *testing.T) {
		s := &ScraperSettings{UseBrowser: true}
		result := s.ShouldUseBrowser(true)
		assert.True(t, result)
	})

	t.Run("global_enabled_use_browser_false", func(t *testing.T) {
		s := &ScraperSettings{UseBrowser: false}
		result := s.ShouldUseBrowser(true)
		assert.False(t, result)
	})
}

func TestScraperSettings_DeepCopy(t *testing.T) {
	t.Parallel()

	t.Run("nil_receiver", func(t *testing.T) {
		var s *ScraperSettings
		result := s.DeepCopy()
		assert.Nil(t, result)
	})

	t.Run("basic_fields", func(t *testing.T) {
		s := &ScraperSettings{
			Enabled:         true,
			Language:        "en",
			Timeout:         30,
			RateLimit:       1000,
			RetryCount:      3,
			UserAgent:       "test-agent",
			UseFlareSolverr: true,
			UseBrowser:      true,
			BaseURL:         "https://example.com",
		}

		result := s.DeepCopy()
		assert.NotNil(t, result)
		assert.Equal(t, s.Enabled, result.Enabled)
		assert.Equal(t, s.Language, result.Language)
		assert.Equal(t, s.Timeout, result.Timeout)
		assert.Equal(t, s.RateLimit, result.RateLimit)
		assert.Equal(t, s.RetryCount, result.RetryCount)
		assert.Equal(t, s.UserAgent, result.UserAgent)
		assert.Equal(t, s.UseFlareSolverr, result.UseFlareSolverr)
		assert.Equal(t, s.UseBrowser, result.UseBrowser)
		assert.Equal(t, s.BaseURL, result.BaseURL)

		result.Enabled = false
		assert.True(t, s.Enabled, "Original should not be modified")
	})

	t.Run("proxy_deep_copy", func(t *testing.T) {
		s := &ScraperSettings{
			Proxy: &ProxyConfig{
				Enabled:        true,
				Profile:        "test-profile",
				DefaultProfile: "default",
				Profiles: map[string]ProxyProfile{
					"profile1": {},
				},
			},
		}

		result := s.DeepCopy()
		assert.NotNil(t, result.Proxy)
		assert.Equal(t, s.Proxy.Enabled, result.Proxy.Enabled)
		assert.Equal(t, s.Proxy.Profile, result.Proxy.Profile)

		result.Proxy.Profile = "modified"
		assert.Equal(t, "test-profile", s.Proxy.Profile, "Original should not be modified")
	})

	t.Run("download_proxy_deep_copy", func(t *testing.T) {
		s := &ScraperSettings{
			DownloadProxy: &ProxyConfig{
				Enabled: true,
				Profile: "dl-profile",
			},
		}

		result := s.DeepCopy()
		assert.NotNil(t, result.DownloadProxy)
		assert.Equal(t, s.DownloadProxy.Profile, result.DownloadProxy.Profile)
	})

	t.Run("cookies_deep_copy", func(t *testing.T) {
		s := &ScraperSettings{
			Cookies: map[string]string{
				"session": "abc123",
				"token":   "xyz789",
			},
		}

		result := s.DeepCopy()
		assert.NotNil(t, result.Cookies)
		assert.Equal(t, len(s.Cookies), len(result.Cookies))
		assert.Equal(t, s.Cookies["session"], result.Cookies["session"])

		result.Cookies["session"] = "modified"
		assert.Equal(t, "abc123", s.Cookies["session"], "Original should not be modified")
	})

	t.Run("extra_deep_copy", func(t *testing.T) {
		s := &ScraperSettings{
			Extra: map[string]any{
				"custom_field": "value",
				"number":       42,
			},
		}

		result := s.DeepCopy()
		assert.NotNil(t, result.Extra)
		assert.Equal(t, len(s.Extra), len(result.Extra))
		assert.Equal(t, s.Extra["custom_field"], result.Extra["custom_field"])
	})

	t.Run("scrape_actress_deep_copy", func(t *testing.T) {
		val := true
		s := &ScraperSettings{
			ScrapeActress: &val,
		}

		result := s.DeepCopy()
		assert.NotNil(t, result.ScrapeActress)
		assert.Equal(t, *s.ScrapeActress, *result.ScrapeActress)

		*result.ScrapeActress = false
		assert.True(t, *s.ScrapeActress, "Original should not be modified")
	})

	t.Run("respect_retry_after_deep_copy", func(t *testing.T) {
		val := false
		s := &ScraperSettings{
			RespectRetryAfter: &val,
		}

		result := s.DeepCopy()
		assert.NotNil(t, result.RespectRetryAfter)
		assert.Equal(t, *s.RespectRetryAfter, *result.RespectRetryAfter)

		*result.RespectRetryAfter = true
		assert.False(t, *s.RespectRetryAfter, "Original should not be modified")
	})
}

func TestScraperSettings_ShouldScrapeActress(t *testing.T) {
	t.Parallel()

	t.Run("nil_receiver", func(t *testing.T) {
		var s *ScraperSettings
		result := s.ShouldScrapeActress(true)
		assert.True(t, result)

		result = s.ShouldScrapeActress(false)
		assert.False(t, result)
	})

	t.Run("per_scraper_override_true", func(t *testing.T) {
		val := true
		s := &ScraperSettings{ScrapeActress: &val}
		result := s.ShouldScrapeActress(false)
		assert.True(t, result, "Override should win over global default")
	})

	t.Run("per_scraper_override_false", func(t *testing.T) {
		val := false
		s := &ScraperSettings{ScrapeActress: &val}
		result := s.ShouldScrapeActress(true)
		assert.False(t, result, "Override should win over global default")
	})

	t.Run("fallback_to_global_default", func(t *testing.T) {
		s := &ScraperSettings{}
		result := s.ShouldScrapeActress(true)
		assert.True(t, result, "Should use global default when no override set")

		result = s.ShouldScrapeActress(false)
		assert.False(t, result)
	})
}

func TestDeepCopyProxyProfiles(t *testing.T) {
	t.Parallel()

	t.Run("nil_input", func(t *testing.T) {
		result := deepCopyProxyProfiles(nil)
		assert.Nil(t, result)
	})

	t.Run("empty_map", func(t *testing.T) {
		result := deepCopyProxyProfiles(map[string]ProxyProfile{})
		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})

	t.Run("multiple_profiles", func(t *testing.T) {
		profiles := map[string]ProxyProfile{
			"profile1": {URL: "http://proxy1:8080", Username: "user1", Password: "pass1"},
			"profile2": {URL: "http://proxy2:8080", Username: "user2", Password: "pass2"},
		}

		result := deepCopyProxyProfiles(profiles)
		assert.NotNil(t, result)
		assert.Equal(t, len(profiles), len(result))
		assert.Equal(t, profiles["profile1"].URL, result["profile1"].URL)
		assert.Equal(t, profiles["profile1"].Username, result["profile1"].Username)

		// Modify copy
		profile := result["profile1"]
		profile.URL = "http://modified:8080"
		result["profile1"] = profile

		assert.Equal(t, "http://proxy1:8080", profiles["profile1"].URL, "Original should not be modified")
	})
}
