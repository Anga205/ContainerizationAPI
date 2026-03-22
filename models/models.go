package models

import (
	"fmt"
	"strings"
	"time"
)

type Request struct {
	Language    string        `json:"language"`         // Programming language (e.g., "c", "python", "javascript")
	Code        string        `json:"code"`             // Source code to execute
	Timeout     time.Duration `json:"timeout"`          // time limit in nanoseconds
	MemoryLimit uint          `json:"max_memory"`       // memory limit in KB
	Stdin       string        `json:"inputs,omitempty"` // input to take from STDIN, optional, can be multiple lines of input
}

type SimpleRequest struct {
	Language    string   `json:"language"`         // Programming language (e.g., "c", "python", "javascript")
	Code        string   `json:"code"`             // Source code to execute
	Timeout     uint     `json:"timeout"`          // time limit in seconds
	MemoryLimit uint     `json:"max_memory"`       // memory limit in KB
	Stdin       []string `json:"inputs,omitempty"` // Array of strings for multiple lines of input, optional
}

func (sr *SimpleRequest) ToRequest() Request {
	return Request{
		Language:    sr.Language,
		Code:        sr.Code,
		Timeout:     time.Duration(sr.Timeout) * time.Second,
		MemoryLimit: sr.MemoryLimit,
		Stdin:       strings.Join(sr.Stdin, "\n"),
	}
}

type Response struct {
	Stdout        string        `json:"output"`
	Stderr        string        `json:"error"`
	ExecutionTime time.Duration `json:"cpu_time"`    // CPU time used by the program in nanoseconds
	MemoryUsed    uint          `json:"memory_used"` // Peak memory used by the program in KB
}

type SimpleResponse struct {
	Stdout        string `json:"output"`
	Stderr        string `json:"error"`
	MemoryUsed    string `json:"memory_used"` // Peak memory used by the program in KB
	ExecutionTime string `json:"cpu_time"`    // CPU time used by the program in seconds (ends in s, e.g., "0.5s")
}

func (r *Response) ToSimpleResponse() SimpleResponse {
	return SimpleResponse{
		Stdout:        r.Stdout,
		Stderr:        r.Stderr,
		ExecutionTime: r.ExecutionTime.String(),
		MemoryUsed:    fmt.Sprintf("%d KB", r.MemoryUsed),
	}
}
