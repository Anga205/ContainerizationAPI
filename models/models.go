package models

import "time"

type Request struct {
	Language    string        `json:"language"`
	Code        string        `json:"code"`
	Timeout     time.Duration `json:"timeout"`
	MemoryLimit uint          `json:"max_memory"`
	Stdin       []string      `json:"inputs,omitempty"`
}

type Response struct {
	Stdout        string        `json:"output"`
	Stderr        string        `json:"error"`
	ExecutionTime time.Duration `json:"cpu_time"`
	MemoryUsed    uint          `json:"memory_used"`
}
