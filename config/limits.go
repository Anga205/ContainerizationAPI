package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type limits struct {
	MaxTimeout         time.Duration `env:"MAX_TIMEOUT" envDefault:"60s"`            // 60 seconds by default
	MaxMemoryLimit     uint          `env:"MAX_MEMORY_LIMIT" envDefault:"131072"`    // 128 MB by default (or 131072 KB)
	DefaultTimeout     time.Duration `env:"DEFAULT_TIMEOUT" envDefault:"5s"`         // 5 seconds by default
	DefaultMemoryLimit uint          `env:"DEFAULT_MEMORY_LIMIT" envDefault:"32768"` // 32 MB by default (or 32768 KB)
}

func (cfg *limits) load() error {
	return env.Parse(cfg)
}
