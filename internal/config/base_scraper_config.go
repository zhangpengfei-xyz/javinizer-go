package config

type BaseScraperConfig struct {
	Enabled           bool               `yaml:"enabled" json:"enabled"`
	RequestDelay      int                `yaml:"request_delay" json:"request_delay"`
	MaxRetries        int                `yaml:"max_retries" json:"max_retries"`
	UserAgent         string             `yaml:"user_agent" json:"user_agent"`
	Proxy             *ProxyConfig       `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	DownloadProxy     *ProxyConfig       `yaml:"download_proxy,omitempty" json:"download_proxy,omitempty"`
	Priority          int                `yaml:"priority" json:"priority"`
	FlareSolverr      FlareSolverrConfig `yaml:"flaresolverr" json:"flaresolverr"`
	RespectRetryAfter *bool              `yaml:"respect_retry_after,omitempty" json:"respect_retry_after,omitempty"`
}

func (c BaseScraperConfig) IsEnabled() bool      { return c.Enabled }
func (c BaseScraperConfig) GetUserAgent() string { return c.UserAgent }
func (c BaseScraperConfig) GetRequestDelay() int { return c.RequestDelay }
func (c BaseScraperConfig) GetMaxRetries() int   { return c.MaxRetries }
func (c BaseScraperConfig) GetProxy() any {
	if c.Proxy == nil {
		return nil
	}
	return c.Proxy
}

func (c BaseScraperConfig) GetDownloadProxy() any {
	if c.DownloadProxy == nil {
		return nil
	}
	return c.DownloadProxy
}

func (c BaseScraperConfig) GetRespectRetryAfter() *bool { return c.RespectRetryAfter }

func ProxyAsConfig(p any) *ProxyConfig {
	if p == nil {
		return nil
	}
	if cfg, ok := p.(*ProxyConfig); ok {
		return cfg
	}
	return nil
}
