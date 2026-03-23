package resourcemanager

import (
	"os"
	"strconv"
	"strings"
)

func readUintFromFile(path string) uint64 {
	value, err := readUintWithError(path)
	if err != nil {
		return 0
	}
	return value
}

func readUintWithError(path string) (uint64, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(content)), 10, 64)
}

func parseMemoryEvents(content string) MemoryEvents {
	events := MemoryEvents{}
	for _, line := range strings.Split(content, "\n") {
		key, value, ok := parseMemoryEventLine(line)
		if !ok {
			continue
		}
		applyMemoryEvent(&events, key, value)
	}
	return events
}

func parseMemoryEventLine(line string) (string, uint64, bool) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return "", 0, false
	}
	value, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return "", 0, false
	}
	return fields[0], value, true
}

func applyMemoryEvent(events *MemoryEvents, key string, value uint64) {
	switch key {
	case "low":
		events.Low = value
	case "high":
		events.High = value
	case "max":
		events.Max = value
	case "oom":
		events.OOM = value
	case "oom_kill":
		events.OOMKill = value
	}
}
