package config

import "github.com/caarlos0/env/v11"

type globals struct {
	RAM_LIMIT           uint `env:"GLOBAL_RAM_LIMIT" envDefault:"1048576"` // 1 GB by default (or 1048576 KB), this is the total amount of RAM that can be used by all running sandboxes at any given time
	ENABLE_QUEUE        bool `env:"ENABLE_QUEUE" envDefault:"false"`       // if false, the server will reject requests when ram limit is reached, if true, the server will queue requests until sufficient ram is available
	ENABLE_DEBUG_ROUTES bool `env:"ENABLE_DEBUG" envDefault:"false"`       // if true, the server will expose debug routes for monitoring and debugging purposes
}

// i know the docs say that queueing is true by default
// ill set it to false by default for now because queueing is not implemented yet

func (cfg *globals) load() error {
	return env.Parse(cfg)
}
