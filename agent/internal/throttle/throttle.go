// Package throttle provides a token-bucket rate limiter for io.Reader/io.Writer.
// Use it to cap backup upload speed during business hours.
package throttle

import (
	"io"
	"time"
)

// Reader wraps an io.Reader and limits read throughput to bytesPerSec bytes/s.
// If bytesPerSec is 0 the reader is a transparent pass-through.
type Reader struct {
	r           io.Reader
	bytesPerSec int64
	bucket      int64     // current tokens
	lastRefill  time.Time
}

// NewReader creates a rate-limited reader.
// mbps is megabytes per second (0 = unlimited).
func NewReader(r io.Reader, mbps float64) *Reader {
	bps := int64(mbps * 1024 * 1024)
	return &Reader{
		r:           r,
		bytesPerSec: bps,
		bucket:      bps,
		lastRefill:  time.Now(),
	}
}

func (tr *Reader) Read(p []byte) (int, error) {
	if tr.bytesPerSec == 0 {
		return tr.r.Read(p)
	}

	tr.refill()

	// How many bytes can we read right now?
	allowed := tr.bucket
	if allowed <= 0 {
		// Sleep until we have at least 1 byte
		sleep := time.Second / time.Duration(tr.bytesPerSec)
		if sleep < time.Millisecond {
			sleep = time.Millisecond
		}
		time.Sleep(sleep)
		tr.refill()
		allowed = tr.bucket
	}

	maxRead := int64(len(p))
	if allowed < maxRead {
		maxRead = allowed
	}

	n, err := tr.r.Read(p[:maxRead])
	tr.bucket -= int64(n)
	return n, err
}

func (tr *Reader) refill() {
	now := time.Now()
	elapsed := now.Sub(tr.lastRefill)
	tr.lastRefill = now

	added := int64(elapsed.Seconds() * float64(tr.bytesPerSec))
	tr.bucket += added
	if tr.bucket > tr.bytesPerSec {
		tr.bucket = tr.bytesPerSec // cap at 1-second burst
	}
}

// Writer wraps an io.Writer and limits write throughput to bytesPerSec bytes/s.
type Writer struct {
	w           io.Writer
	bytesPerSec int64
	bucket      int64
	lastRefill  time.Time
}

// NewWriter creates a rate-limited writer. mbps is megabytes per second (0 = unlimited).
func NewWriter(w io.Writer, mbps float64) *Writer {
	bps := int64(mbps * 1024 * 1024)
	return &Writer{
		w:           w,
		bytesPerSec: bps,
		bucket:      bps,
		lastRefill:  time.Now(),
	}
}

func (tw *Writer) Write(p []byte) (int, error) {
	if tw.bytesPerSec == 0 {
		return tw.w.Write(p)
	}

	written := 0
	for written < len(p) {
		tw.refill()
		if tw.bucket <= 0 {
			sleep := time.Second / time.Duration(tw.bytesPerSec)
			if sleep < time.Millisecond {
				sleep = time.Millisecond
			}
			time.Sleep(sleep)
			tw.refill()
		}

		chunk := int(tw.bucket)
		if chunk > len(p)-written {
			chunk = len(p) - written
		}

		n, err := tw.w.Write(p[written : written+chunk])
		tw.bucket -= int64(n)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

func (tw *Writer) refill() {
	now := time.Now()
	elapsed := now.Sub(tw.lastRefill)
	tw.lastRefill = now

	added := int64(elapsed.Seconds() * float64(tw.bytesPerSec))
	tw.bucket += added
	if tw.bucket > tw.bytesPerSec {
		tw.bucket = tw.bytesPerSec
	}
}
