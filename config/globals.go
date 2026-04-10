package config

import "github.com/caarlos0/env/v11"

type globals struct {
	RAM_LIMIT           uint `env:"GLOBAL_RAM_LIMIT" envDefault:"1048576" json:"ram_limit"`     // 1 GB by default (or 1048576 KB), this is the total amount of RAM that can be used by all running sandboxes at any given time
	ENABLE_DEBUG_ROUTES bool `env:"ENABLE_DEBUG" envDefault:"false" json:"enable_debug_routes"` // if true, the server will expose debug routes for monitoring and debugging purposes
}

func (cfg *globals) load() error {
	return env.Parse(cfg)
}
