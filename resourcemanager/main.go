package resourcemanager

import (
	"CodeSandboxAPI/config"
	"sync"
)

var (
	ram_available uint = config.Config.Globals.RAM_LIMIT
	ram_lock      sync.RWMutex
)
