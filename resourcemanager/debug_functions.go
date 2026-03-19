package resourcemanager

import "CodeSandboxAPI/config"

func GetRamUsed() uint {
	ram_lock.RLock()
	ramUsed := config.Config.Globals.RAM_LIMIT - ram_available
	ram_lock.RUnlock()
	return ramUsed
}

func GetRamAvailable() uint {
	ram_lock.RLock()
	ramAvailable := ram_available
	ram_lock.RUnlock()
	return ramAvailable
}
