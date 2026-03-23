package resourcemanager

import "fmt"

func memoryLimitBytes(memoryLimitKB uint) string {
	if memoryLimitKB == 0 {
		return "max\n"
	}
	return fmt.Sprintf("%d\n", uint64(memoryLimitKB)*1024)
}

func cgroupControlFiles(memoryLimitKB uint) map[string]string {
	limit := memoryLimitBytes(memoryLimitKB)
	return map[string]string{
		"cpu.max":          "100000 100000\n",
		"memory.max":       limit,
		"memory.high":      limit,
		"memory.swap.max":  "0\n",
		"memory.oom.group": "1\n",
		"pids.max":         "64\n",
	}
}
