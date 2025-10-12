package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Result struct {
	Status     string        // "ok" | "error" | "skipped"
	Duration   time.Duration
	Error      string
	ResultFile string        // Path to result file (if created)
}

func RunJob(ctx context.Context, jobType string, argsRaw json.RawMessage, timeoutSec int) Result {
	dl := time.Duration(timeoutSec) * time.Second
	if dl <= 0 {
		dl = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, dl)
	defer cancel()

	start := time.Now()
	res := Result{Status: "ok"}

	switch jobType {
	case "sleep":
		var a struct{ Seconds int `json:"seconds"` }
		json.Unmarshal(argsRaw, &a)
		if a.Seconds <= 0 {
			a.Seconds = 5
		}
		t := time.NewTimer(time.Duration(a.Seconds) * time.Second)
		select {
		case <-ctx.Done():
			res.Status = "error"
			res.Error = "timeout/cancelled"
		case <-t.C:
			// ok - create a simple result file
			resultFile, err := createSleepResult(a.Seconds)
			if err == nil {
				res.ResultFile = resultFile
			}
		}

	case "hash":
		// Simple CPU canary: hash random-ish buffers for N seconds
		var a struct{ Seconds int `json:"seconds"` }
		json.Unmarshal(argsRaw, &a)
		if a.Seconds <= 0 {
			a.Seconds = 10
		}
		buf := make([]byte, 1<<16)
		for i := range buf {
			buf[i] = byte(i)
		}

		// Create result file to store hashes
		resultFile, err := os.CreateTemp("", "hash-result-*.txt")
		if err == nil {
			defer resultFile.Close()
			res.ResultFile = resultFile.Name()

			hashCount := 0
			for time.Since(start) < time.Duration(a.Seconds)*time.Second {
				select {
				case <-ctx.Done():
					res.Status = "error"
					res.Error = "timeout/cancelled"
					goto done
				default:
					h := sha256.Sum256(buf)
					hashHex := hex.EncodeToString(h[:])
					fmt.Fprintf(resultFile, "%s\n", hashHex)
					hashCount++
				}
			}
			fmt.Fprintf(resultFile, "\nTotal hashes computed: %d\n", hashCount)
			fmt.Fprintf(resultFile, "Duration: %v\n", time.Since(start))
		}
	done:

	default:
		res.Status = "skipped"
		res.Error = "unsupported job type"
	}

	res.Duration = time.Since(start)
	return res
}

// createSleepResult creates a simple result file for sleep jobs
func createSleepResult(seconds int) (string, error) {
	tmpFile, err := os.CreateTemp("", "sleep-result-*.txt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	content := fmt.Sprintf("Sleep job completed successfully\nDuration: %d seconds\nCompleted at: %s\n",
		seconds, time.Now().Format(time.RFC3339))
	
	if _, err := tmpFile.WriteString(content); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

var ErrTimeout = errors.New("timeout")