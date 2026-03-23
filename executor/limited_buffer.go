package executor

import "bytes"

// limitedBuffer stores up to max bytes and discards the rest without failing writes.
type limitedBuffer struct {
	buf bytes.Buffer
	max int
}

func newLimitedBuffer(max int) *limitedBuffer {
	return &limitedBuffer{max: max}
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.max <= 0 {
		return len(p), nil
	}

	remaining := l.max - l.buf.Len()
	if remaining > 0 {
		if len(p) <= remaining {
			_, _ = l.buf.Write(p)
		} else {
			_, _ = l.buf.Write(p[:remaining])
		}
	}
	return len(p), nil
}

func (l *limitedBuffer) String() string {
	return l.buf.String()
}
