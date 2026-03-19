package resourcemanager

func ReserveRAM(amount uint) bool {
	ram_lock.Lock()
	if amount > ram_available {
		ram_lock.Unlock()
		return false
	}
	ram_available -= amount
	ram_lock.Unlock()
	return true
}

func ReleaseRAM(amount uint) {
	ram_lock.Lock()
	ram_available += amount
	ram_lock.Unlock()
}
