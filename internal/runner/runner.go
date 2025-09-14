package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

type Result struct {
	Status   string        // "ok" | "error" | "skipped"
	Duration time.Duration
	Error    string
}

func RunJob(ctx context.Context, jobType string, argsRaw json.RawMessage, timeoutSec int) Result {
	dl := time.Duration(timeoutSec) * time.Second
	if dl <= 0 { dl = 30 * time.Second }
	ctx, cancel := context.WithTimeout(ctx, dl)
	defer cancel()

	start := time.Now()
	res := Result{Status: "ok"}

	switch jobType {
	case "sleep":
		var a struct{ Seconds int `json:"seconds"` }
		json.Unmarshal(argsRaw, &a)
		if a.Seconds <= 0 { a.Seconds = 5 }
		t := time.NewTimer(time.Duration(a.Seconds) * time.Second)
		select {
		case <-ctx.Done():
			res.Status = "error"; res.Error = "timeout/cancelled"
		case <-t.C:
			// ok
		}

	case "hash":
		// Simple CPU canary: hash random-ish buffers for N seconds
		var a struct{ Seconds int `json:"seconds"` }
		json.Unmarshal(argsRaw, &a)
		if a.Seconds <= 0 { a.Seconds = 10 }
		buf := make([]byte, 1<<16)
		for i := range buf { buf[i] = byte(i) }
		for time.Since(start) < time.Duration(a.Seconds)*time.Second {
			select {
			case <-ctx.Done():
				res.Status = "error"; res.Error = "timeout/cancelled"
				break
			default:
				h := sha256.Sum256(buf)
				_ = hex.EncodeToString(h[:])
			}
		}

	default:
		res.Status = "skipped"
		res.Error = "unsupported job type"
	}

	res.Duration = time.Since(start)
	return res
}

var ErrTimeout = errors.New("timeout")
