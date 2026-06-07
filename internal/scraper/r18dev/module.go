package r18dev

import (
	"github.com/javinizer/javinizer-go/internal/config"
	"github.com/javinizer/javinizer-go/internal/database"
	"github.com/javinizer/javinizer-go/internal/models"
	"github.com/javinizer/javinizer-go/internal/scraperutil"
)

func init() {
	m := &scraperModule{}
	m.StandardModule = scraperutil.StandardModule{
		ScraperName:        "r18dev",
		ScraperDescription: "R18.dev",
		ScraperOptions: []any{
			models.ScraperOption{
				Key:         "language",
				Label:       "Language",
				Description: "Language for metadata fields",
				Type:        "select",
				Default:     "en",
				Choices: []models.ScraperChoice{
					{Value: "en", Label: "English"},
					{Value: "ja", Label: "Japanese"},
				},
			},
			models.ScraperOption{
				Key:         "placeholder_threshold",
				Label:       "Placeholder Threshold",
				Description: "File size threshold in KB for detecting placeholder screenshots. Files smaller than this are checked against known placeholder hashes.",
				Type:        "number",
				Default:     10,
				Min:         scraperutil.IntPtr(1),
				Max:         scraperutil.IntPtr(1000),
				Unit:        "KB",
			},
			models.ScraperOption{
				Key:         "extra_placeholder_hashes",
				Label:       "Extra Placeholder Hashes",
				Description: "Additional SHA256 hashes of known placeholder images. Each hash is a 64-character hex string.",
				Type:        "string",
			},
			models.ScraperOption{
				Key:         "respect_retry_after",
				Label:       "Respect Retry-After",
				Description: "Respect the Retry-After header from Cloudflare on 429 rate-limit responses. When enabled, waits the server-specified duration before retrying instead of using only exponential backoff.",
				Type:        "boolean",
				Default:     true,
			},
		},
		ScraperDefaults: config.ScraperSettings{
			Enabled:  true,
			Language: "en",
		},
		ScraperPriority: 100,
		ConfigType:      func() scraperutil.ScraperConfigInterface { return &R18DevConfig{} },
		NewScraperFunc: func(settings config.ScraperSettings, db *database.DB, globalConfig *config.ScrapersConfig) (models.Scraper, error) {
			var globalProxy *config.ProxyConfig
			var globalFlareSolverr config.FlareSolverrConfig
			if globalConfig != nil {
				globalProxy = &globalConfig.Proxy
				globalFlareSolverr = globalConfig.FlareSolverr
			}
			return New(settings, globalProxy, globalFlareSolverr), nil
		},
		FlatBuilder: func(fc *scraperutil.FlattenedConfig, overrides scraperutil.FlattenOverrides) any {
			return &config.ScraperSettings{
				Enabled:           fc.Enabled,
				Language:          overrides.Language,
				RateLimit:         fc.RateLimit,
				RetryCount:        fc.MaxRetries,
				Proxy:             config.ProxyAsConfig(fc.Proxy),
				DownloadProxy:     config.ProxyAsConfig(fc.DownloadProxy),
				RespectRetryAfter: fc.RespectRetryAfter,
			}
		},
	}
	scraperutil.RegisterModule(m)
}

type scraperModule struct {
	scraperutil.StandardModule
}

var _ scraperutil.ScraperModule = (*scraperModule)(nil)
