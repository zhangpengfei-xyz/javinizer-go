package scraperutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testConfig struct {
	BaseScraperConfigEmbed
	Language string `yaml:"language" json:"language"`
}

type BaseScraperConfigEmbed struct {
	Enabled      bool
	RequestDelay int
	MaxRetries   int
	UserAgent    string
}

func (c *testConfig) IsEnabled() bool             { return c.Enabled }
func (c *testConfig) GetUserAgent() string        { return c.UserAgent }
func (c *testConfig) GetRequestDelay() int        { return c.RequestDelay }
func (c *testConfig) GetMaxRetries() int          { return c.MaxRetries }
func (c *testConfig) GetProxy() any               { return nil }
func (c *testConfig) GetDownloadProxy() any       { return nil }
func (c *testConfig) GetRespectRetryAfter() *bool { return nil }

func (c *testConfig) ValidateConfig(a any) error {
	return nil
}

func TestStandardModule_SatisfiesInterface(t *testing.T) {
	var _ ScraperModule = (*StandardModule)(nil)
}

func TestStandardModule_Name(t *testing.T) {
	m := &StandardModule{ScraperName: "test"}
	assert.Equal(t, "test", m.Name())
}

func TestStandardModule_Description(t *testing.T) {
	m := &StandardModule{ScraperDescription: "Test Scraper"}
	assert.Equal(t, "Test Scraper", m.Description())
}

func TestStandardModule_Options(t *testing.T) {
	opts := []any{"opt1", "opt2"}
	m := &StandardModule{ScraperOptions: opts}
	assert.Equal(t, opts, m.Options())
}

func TestStandardModule_Defaults(t *testing.T) {
	defaults := map[string]any{"enabled": true}
	m := &StandardModule{ScraperDefaults: defaults}
	assert.Equal(t, defaults, m.Defaults())
}

func TestStandardModule_Priority(t *testing.T) {
	m := &StandardModule{ScraperPriority: 100}
	assert.Equal(t, 100, m.Priority())
}

func TestStandardModule_Constructor(t *testing.T) {
	constructor := func() {}
	m := &StandardModule{NewScraperFunc: constructor}
	result := m.Constructor()
	assert.NotNil(t, result)
}

func TestStandardModule_Constructor_Nil(t *testing.T) {
	m := &StandardModule{}
	result := m.Constructor()
	assert.Nil(t, result)
}

func TestStandardModule_Validator(t *testing.T) {
	m := &StandardModule{
		ConfigType: func() ScraperConfigInterface { return &testConfig{} },
	}

	result := m.Validator()
	assert.NotNil(t, result)

	vf, ok := result.(ValidatorFunc)
	assert.True(t, ok)
	assert.NotNil(t, vf)
}

func TestStandardModule_ConfigFactory(t *testing.T) {
	m := &StandardModule{
		ConfigType: func() ScraperConfigInterface { return &testConfig{} },
	}

	result := m.ConfigFactory()
	assert.NotNil(t, result)

	cf, ok := result.(ConfigFactory)
	assert.True(t, ok)

	cfg := cf()
	assert.NotNil(t, cfg)

	tc, ok := cfg.(*testConfig)
	assert.True(t, ok)
	assert.NotNil(t, tc)
}

func TestStandardModule_FlattenFunc_WithBuilder(t *testing.T) {
	builder := func(fc *FlattenedConfig, overrides FlattenOverrides) any {
		return map[string]any{"enabled": fc.Enabled, "rate_limit": fc.RateLimit}
	}

	m := &StandardModule{
		FlatBuilder: builder,
	}

	result := m.FlattenFunc()
	assert.NotNil(t, result)

	ff, ok := result.(FlattenFunc)
	assert.True(t, ok)
	assert.NotNil(t, ff)
}

func TestStandardModule_FlattenFunc_WithRawBuilder(t *testing.T) {
	builderRaw := func(fc *FlattenedConfig, overrides FlattenOverrides, raw any) any {
		return map[string]any{"enabled": fc.Enabled, "rate_limit": fc.RateLimit}
	}

	m := &StandardModule{
		FlatBuilderRaw: builderRaw,
		UseRawBuilder:  true,
	}

	result := m.FlattenFunc()
	assert.NotNil(t, result)

	ff, ok := result.(FlattenFunc)
	assert.True(t, ok)
	assert.NotNil(t, ff)
}

func TestStandardModule_FlattenFunc_NilWithoutBuilder(t *testing.T) {
	m := &StandardModule{}
	result := m.FlattenFunc()
	assert.Nil(t, result)
}

func TestStandardModule_EmbedOverride(t *testing.T) {
	type customModule struct {
		StandardModule
	}

	m := &customModule{}
	m.StandardModule = StandardModule{
		ScraperName:        "custom",
		ScraperDescription: "Custom Scraper",
		ScraperPriority:    50,
		ConfigType:         func() ScraperConfigInterface { return &testConfig{} },
	}

	var _ ScraperModule = m
	assert.Equal(t, "custom", m.Name())
	assert.Equal(t, "Custom Scraper", m.Description())
	assert.Equal(t, 50, m.Priority())
}
