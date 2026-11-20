package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"
)

// Every metadata request is retried on network errors (no HTTP response)
// until the context is done, sleeping retryBackoff between attempts (the last
// entry repeats). Vars so tests can fake time.
var (
	budget       = 30 * time.Second
	retryBackoff = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 5 * time.Second}
	now          = time.Now
	sleep        = func(ctx context.Context, d time.Duration) {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-ctx.Done():
		case <-t.C:
		}
	}
	httpClient = &http.Client{Timeout: 2 * time.Second}
)

// statusError is an HTTP answer other than 200; it is final and never retried
type statusError struct{ Code int }

func (e *statusError) Error() string { return fmt.Sprintf("Status not ok: %d", e.Code) }

// withBudget returns a context that ends one retry budget from now
func withBudget(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithDeadline(parent, now().Add(budget))
}

// retry calls try until it returns nil or a *statusError, or ctx is done; any other error
// is a network error and is retried with retryBackoff, logging what on each failed attempt.
// Without a deadline only cancellation ends it: one missed reply never rules a provider out.
func retry(ctx context.Context, what string, try func() error) error {
	deadline, hasDeadline := ctx.Deadline()
	for attempt := 1; ; attempt++ {
		err := try()
		var se *statusError
		if err == nil || errors.As(err, &se) {
			return err
		}
		if ctx.Err() != nil {
			return fmt.Errorf("Could not contact metadata service: %w (%w)", err, ctx.Err())
		}
		left := time.Duration(math.MaxInt64)
		if hasDeadline {
			left = deadline.Sub(now())
		}
		if left <= 0 {
			return fmt.Errorf("Could not contact metadata service: %w", err)
		}
		delay := min(retryBackoff[min(attempt, len(retryBackoff))-1], left)
		log.Printf("%s attempt %d failed: %s, retrying in %s (%s left)", what, attempt, err,
			delay.Round(100*time.Millisecond), left.Round(100*time.Millisecond))
		sleep(ctx, delay)
	}
}

// fetch sends a bodyless request, retrying network errors; any status other than 200 is a *statusError
func fetch(ctx context.Context, method, url string, hdr http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	if hdr != nil {
		req.Header = hdr
	}
	var body []byte
	err = retry(ctx, method+" "+url, func() error {
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return &statusError{resp.StatusCode}
		}
		body, err = io.ReadAll(resp.Body)
		return err
	})
	return body, err
}

// get fetches url with GET (see fetch)
func get(ctx context.Context, url string, hdr http.Header) ([]byte, error) {
	return fetch(ctx, http.MethodGet, url, hdr)
}
