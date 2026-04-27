package imageproc

import (
	"path/filepath"
	"strings"
)

// Config controls card normalization and optional debug dumps (YAML: imageproc.*).
type Config struct {
	CardNormalize        bool
	StrictCardNormalize  bool
	MaxWorkingLongEdge   int
	WarpWidth            int
	MinQuadAreaRatio     float64
	MaxQuadAreaRatio     float64
	DebugNormalize       bool
	DebugNormalizeOutDir string
}

// DefaultConfig returns conservative defaults: normalization on, non-strict fallback.
func DefaultConfig() Config {
	return Config{
		CardNormalize:        true,
		StrictCardNormalize:  false,
		MaxWorkingLongEdge:   960,
		WarpWidth:            800,
		MinQuadAreaRatio:     0.12,
		MaxQuadAreaRatio:     0.92,
		DebugNormalize:       false,
		DebugNormalizeOutDir: "",
	}
}

// ValidateDebug returns ErrInvalidDebugOutputDir when debug is enabled without a usable directory.
func ValidateDebug(c Config) error {
	if !c.DebugNormalize {
		return nil
	}
	if strings.TrimSpace(c.DebugNormalizeOutDir) == "" {
		return ErrInvalidDebugOutputDir
	}
	_ = filepath.Clean(c.DebugNormalizeOutDir)
	return nil
}
