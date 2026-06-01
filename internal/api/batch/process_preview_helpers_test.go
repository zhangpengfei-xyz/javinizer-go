package batch

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javinizer/javinizer-go/internal/config"
	"github.com/javinizer/javinizer-go/internal/logging"
	"github.com/javinizer/javinizer-go/internal/models"
	"github.com/javinizer/javinizer-go/internal/template"
	"github.com/javinizer/javinizer-go/internal/worker"
)

func newTrailerTestMovie() *models.Movie {
	return &models.Movie{
		ID:         "ABC-123",
		Title:      "Sample Title",
		Maker:      "IdeaPocket",
		TrailerURL: "https://example.com/ABC-123-trailer.mp4",
	}
}

func newTrailerTestFileResults() []*worker.FileResult {
	return []*worker.FileResult{
		{FilePath: "/videos/ABC-123.mp4"},
	}
}

func newTrailerTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Output.FolderFormat = "<ID>"
	cfg.Output.FileFormat = "<ID>"
	cfg.Output.SubfolderFormat = []string{}
	cfg.Output.TrailerFormat = "<ID>-trailer.mp4"
	cfg.Output.DownloadTrailer = true
	cfg.Output.DownloadExtrafanart = false
	cfg.Output.MaxPathLength = 0
	return cfg
}

func TestGenerateTrailerPath_EnabledWithURL(t *testing.T) {
	cfg := newTrailerTestConfig()
	movie := newTrailerTestMovie()
	fileResults := newTrailerTestFileResults()
	ctx := template.NewContextFromMovie(movie)
	engine := template.NewEngine()
	folderPath := filepath.Join("/library", "ABC-123")

	got := generateTrailerPath(movie, fileResults, cfg, ctx, engine, folderPath)

	want := filepath.Join(folderPath, "ABC-123-trailer.mp4")
	assert.Equal(t, want, got)
}

func TestGenerateTrailerPath_DisabledInConfig(t *testing.T) {
	cfg := newTrailerTestConfig()
	cfg.Output.DownloadTrailer = false
	movie := newTrailerTestMovie()
	fileResults := newTrailerTestFileResults()
	ctx := template.NewContextFromMovie(movie)
	engine := template.NewEngine()
	folderPath := filepath.Join("/library", "ABC-123")

	got := generateTrailerPath(movie, fileResults, cfg, ctx, engine, folderPath)

	assert.Equal(t, "", got)
}

func TestGenerateTrailerPath_NoTrailerURL(t *testing.T) {
	cfg := newTrailerTestConfig()
	movie := newTrailerTestMovie()
	movie.TrailerURL = ""
	fileResults := newTrailerTestFileResults()
	ctx := template.NewContextFromMovie(movie)
	engine := template.NewEngine()
	folderPath := filepath.Join("/library", "ABC-123")

	got := generateTrailerPath(movie, fileResults, cfg, ctx, engine, folderPath)

	assert.Equal(t, "", got)
}

func TestGenerateTrailerPath_NilMovie(t *testing.T) {
	cfg := newTrailerTestConfig()
	fileResults := newTrailerTestFileResults()
	engine := template.NewEngine()
	folderPath := filepath.Join("/library", "ABC-123")
	var movie *models.Movie

	got := generateTrailerPath(movie, fileResults, cfg, nil, engine, folderPath)

	assert.Equal(t, "", got)
}

func TestGenerateTrailerPath_EmptyFormatFallsBackToIDTrailer(t *testing.T) {
	cfg := newTrailerTestConfig()
	cfg.Output.TrailerFormat = ""
	movie := newTrailerTestMovie()
	fileResults := newTrailerTestFileResults()
	ctx := template.NewContextFromMovie(movie)
	engine := template.NewEngine()
	folderPath := filepath.Join("/library", "ABC-123")

	got := generateTrailerPath(movie, fileResults, cfg, ctx, engine, folderPath)

	want := filepath.Join(folderPath, "ABC-123-trailer.mp4")
	assert.Equal(t, want, got)
}

func TestGenerateTrailerPath_MultiPartIncludesPartSuffix(t *testing.T) {
	cfg := newTrailerTestConfig()
	cfg.Output.TrailerFormat = "<ID>-<PARTSUFFIX>-trailer.mp4"
	movie := newTrailerTestMovie()
	fileResults := []*worker.FileResult{
		{FilePath: "/videos/ABC-123-pt1.mp4", IsMultiPart: true, PartNumber: 1, PartSuffix: "-pt1"},
		{FilePath: "/videos/ABC-123-pt2.mp4", IsMultiPart: true, PartNumber: 2, PartSuffix: "-pt2"},
	}
	ctx := template.NewContextFromMovie(movie)
	engine := template.NewEngine()
	folderPath := filepath.Join("/library", "ABC-123")

	got := generateTrailerPath(movie, fileResults, cfg, ctx, engine, folderPath)

	want := filepath.Join(folderPath, "ABC-123--pt1-trailer.mp4")
	assert.Equal(t, want, got)
}

func TestGenerateTrailerPath_BlankIDFallsBackToDashTrailer(t *testing.T) {
	cfg := newTrailerTestConfig()
	cfg.Output.TrailerFormat = ""
	movie := &models.Movie{
		ID:         "",
		TrailerURL: "https://example.com/x.mp4",
	}
	fileResults := newTrailerTestFileResults()
	ctx := template.NewContextFromMovie(movie)
	engine := template.NewEngine()
	folderPath := filepath.Join("/library", "ABC-123")

	got := generateTrailerPath(movie, fileResults, cfg, ctx, engine, folderPath)

	want := filepath.Join(folderPath, "-trailer.mp4")
	assert.Equal(t, want, got)
}

func TestValidatePathLengths_TrailerPathOverflow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.MaxPathLength = 10
	engine := template.NewEngine()
	longTrailer := "/very/long/path/to/the/folder/that/exceeds/limit/IPX-535-trailer.mp4"

	var buf bytes.Buffer
	logger := logging.L()
	prevOut := logger.Out
	prevFormatter := logger.Formatter
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.TextFormatter{DisableColors: true, DisableTimestamp: true})
	t.Cleanup(func() {
		logger.SetOutput(prevOut)
		logger.SetFormatter(prevFormatter)
	})

	validatePathLengths(cfg, engine, nil, "", nil, "", "", "", nil, longTrailer)

	log := buf.String()
	assert.Contains(t, log, "trailer path exceeds max length")
	assert.Contains(t, log, "IPX-535-trailer.mp4")
	assert.Contains(t, log, "max: 10")
}

func TestGeneratePreview_IncludesTrailerPath(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.FolderFormat = "<ID>"
	cfg.Output.FileFormat = "<ID>"
	cfg.Output.SubfolderFormat = []string{}
	cfg.Output.TrailerFormat = "<ID>-trailer.mp4"
	cfg.Output.DownloadTrailer = true
	cfg.Metadata.NFO.FilenameTemplate = "<ID>.nfo"

	movie := &models.Movie{
		ID:         "DEF-456",
		TrailerURL: "https://example.com/DEF-456-trailer.mp4",
	}
	fileResults := []*worker.FileResult{
		{FilePath: "/videos/DEF-456.mp4"},
	}

	resp := generatePreview(movie, fileResults, "/library", cfg, "", false, false)

	folderPath := filepath.Join("/library", "DEF-456")
	want := filepath.Join(folderPath, "DEF-456-trailer.mp4")
	require.Equal(t, want, resp.TrailerPath)
}

func TestGeneratePreview_ExcludesTrailerWhenDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.FolderFormat = "<ID>"
	cfg.Output.FileFormat = "<ID>"
	cfg.Output.SubfolderFormat = []string{}
	cfg.Output.TrailerFormat = "<ID>-trailer.mp4"
	cfg.Output.DownloadTrailer = false
	cfg.Metadata.NFO.FilenameTemplate = "<ID>.nfo"

	movie := &models.Movie{
		ID:         "GHI-789",
		TrailerURL: "https://example.com/GHI-789-trailer.mp4",
	}
	fileResults := []*worker.FileResult{
		{FilePath: "/videos/GHI-789.mp4"},
	}

	resp := generatePreview(movie, fileResults, "/library", cfg, "", false, false)

	assert.Equal(t, "", resp.TrailerPath)
}

func TestGeneratePreview_ExcludesTrailerWhenNoURL(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.FolderFormat = "<ID>"
	cfg.Output.FileFormat = "<ID>"
	cfg.Output.SubfolderFormat = []string{}
	cfg.Output.TrailerFormat = "<ID>-trailer.mp4"
	cfg.Output.DownloadTrailer = true
	cfg.Metadata.NFO.FilenameTemplate = "<ID>.nfo"

	movie := &models.Movie{
		ID:         "JKL-012",
		TrailerURL: "",
	}
	fileResults := []*worker.FileResult{
		{FilePath: "/videos/JKL-012.mp4"},
	}

	resp := generatePreview(movie, fileResults, "/library", cfg, "", false, false)

	assert.Equal(t, "", resp.TrailerPath)
}

func TestGeneratePreview_SkipDownloadSuppressesTrailer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.FolderFormat = "<ID>"
	cfg.Output.FileFormat = "<ID>"
	cfg.Output.SubfolderFormat = []string{}
	cfg.Output.TrailerFormat = "<ID>-trailer.mp4"
	cfg.Output.DownloadTrailer = true
	cfg.Metadata.NFO.FilenameTemplate = "<ID>.nfo"

	movie := &models.Movie{
		ID:         "MNO-345",
		TrailerURL: "https://example.com/MNO-345-trailer.mp4",
	}
	fileResults := []*worker.FileResult{
		{FilePath: "/videos/MNO-345.mp4"},
	}

	resp := generatePreview(movie, fileResults, "/library", cfg, "", false, true)

	assert.Equal(t, "", resp.TrailerPath)
	assert.Equal(t, "", resp.PosterPath)
	assert.Equal(t, "", resp.FanartPath)
}
