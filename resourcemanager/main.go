package resourcemanager

import (
	"CodeSandboxAPI/config"
	"sync"
)

var (
	ram_available uint = config.Config.Globals.RAM_LIMIT
	ram_lock      sync.RWMutex
)

func ResetRAMForTests(limit uint) {
	ram_lock.Lock()
	ram_available = limit
	ram_lock.Unlock()
}
