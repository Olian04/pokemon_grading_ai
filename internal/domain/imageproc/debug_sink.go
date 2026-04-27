package imageproc

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// DebugSink writes PNG snapshots when normalization debug is enabled.
type DebugSink struct {
	enabled bool
	baseDir string
	prefix  string
	step    atomic.Uint32
}

func newDebugSink(cfg Config, side string) (*DebugSink, error) {
	if !cfg.DebugNormalize {
		return &DebugSink{}, nil
	}
	if err := ValidateDebug(cfg); err != nil {
		return nil, err
	}
	ts := time.Now().UTC().Format("20060102T150405.000000000")
	dir := filepath.Join(cfg.DebugNormalizeOutDir, ts+"_"+side)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("imageproc debug mkdir: %w", err)
	}
	return &DebugSink{enabled: true, baseDir: dir, prefix: ""}, nil
}

func (d *DebugSink) dir() string {
	if d == nil {
		return ""
	}
	return d.baseDir
}

func (d *DebugSink) Write(stepName string, img image.Image) error {
	if d == nil {
		return nil
	}
	if !d.enabled || img == nil {
		return nil
	}
	n := d.step.Add(1)
	name := fmt.Sprintf("%02d_%s.png", n, stepName)
	path := filepath.Join(d.baseDir, name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
