package models

import "time"

type Request struct {
	Language    string        `json:"language"`         // Programming language (e.g., "c", "python", "javascript")
	Code        string        `json:"code"`             // Source code to execute
	Timeout     time.Duration `json:"timeout"`          // time limit in seconds
	MemoryLimit uint          `json:"max_memory"`       // memory limit in KB
	Stdin       []string      `json:"inputs,omitempty"` // Array of strings for multiple lines of input, optional
}

type Response struct {
	Stdout        string        `json:"output"`
	Stderr        string        `json:"error"`
	ExecutionTime time.Duration `json:"cpu_time"`
	MemoryUsed    uint          `json:"memory_used"`
}
