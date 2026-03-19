package models

import "time"

type Request struct {
	Language    string
	Code        string
	Timeout     time.Duration
	MemoryLimit uint
	Stdin       string
}

type Response struct {
	Stdout        string
	Stderr        string
	ExecutionTime time.Duration
	MemoryUsed    uint
}
