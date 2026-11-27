package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// drop closes the connection without a response, i.e. a network-level failure
func drop(t *testing.T, w http.ResponseWriter) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		t.Errorf("hijack: %v", err)
		return
	}
	conn.Close()
}

// flaky drops the first n requests, then serves h; the count includes drops
func flaky(t *testing.T, n int32, h http.HandlerFunc) (*httptest.Server, *atomic.Int32) {
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) <= n {
			drop(t, w)
			return
		}
		h(w, r)
	}))
	// fresh connection per request, so net/http never transparently retries
	srv.Config.SetKeepAlivesEnabled(false)
	t.Cleanup(srv.Close)
	return srv, &count
}

func ok(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) }

// clock and sleeps are the fake time and retry sleeps since the last fakeTime
var (
	clockMu sync.Mutex
	clock   time.Time
	sleeps  []time.Duration
)

// fakeTime replaces now and sleep with a clock that only sleep advances. It
// starts at the real time so withBudget's context deadline stays in the future.
func fakeTime(t *testing.T) {
	clockMu.Lock()
	clock, sleeps = time.Now(), nil
	clockMu.Unlock()
	oldNow, oldSleep := now, sleep
	now = func() time.Time {
		clockMu.Lock()
		defer clockMu.Unlock()
		return clock
	}
	sleep = func(_ context.Context, d time.Duration) {
		clockMu.Lock()
		defer clockMu.Unlock()
		sleeps = append(sleeps, d)
		clock = clock.Add(d)
	}
	t.Cleanup(func() { now, sleep = oldNow, oldSleep })
}

func checkSleeps(t *testing.T, want ...time.Duration) {
	t.Helper()
	clockMu.Lock()
	defer clockMu.Unlock()
	if !reflect.DeepEqual(sleeps, want) {
		t.Fatalf("want sleeps %v, got %v", want, sleeps)
	}
}

func TestGetRetriesNetworkErrors(t *testing.T) {
	fakeTime(t)
	srv, count := flaky(t, 2, ok)
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	body, err := get(ctx, srv.URL, nil)
	if err != nil || string(body) != "ok" {
		t.Fatalf("get: %q, %v", body, err)
	}
	if count.Load() != 3 {
		t.Fatalf("want 3 requests, got %d", count.Load())
	}
	checkSleeps(t, time.Second, 2*time.Second)
}

func TestGetStatusNotRetried(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusMethodNotAllowed} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			fakeTime(t)
			srv, count := flaky(t, 0, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) })
			ctx, cancel := withBudget(context.Background())
			defer cancel()
			_, err := get(ctx, srv.URL, nil)
			var se *statusError
			if !errors.As(err, &se) || se.Code != code {
				t.Fatalf("want statusError %d, got %v", code, err)
			}
			if count.Load() != 1 {
				t.Fatalf("want 1 request, got %d", count.Load())
			}
			checkSleeps(t)
		})
	}
}

func TestGetDeadline(t *testing.T) {
	fakeTime(t)
	srv, count := flaky(t, 1<<30, ok)
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	if _, err := get(ctx, srv.URL, nil); err == nil {
		t.Fatal("get succeeded with every request dropped")
	}
	if count.Load() < 9 {
		t.Fatalf("want at least 9 requests, got %d", count.Load())
	}
	s := time.Second
	// 1+2+4+5*4 = 27s, then the last sleep is clamped to the 3s left
	checkSleeps(t, s, 2*s, 4*s, 5*s, 5*s, 5*s, 5*s, 3*s)
}

func TestGetCancelled(t *testing.T) {
	fakeTime(t)
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	srv, count := flaky(t, 0, func(w http.ResponseWriter, _ *http.Request) {
		cancel()
		drop(t, w)
	})
	if _, err := get(ctx, srv.URL, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if count.Load() != 1 {
		t.Fatalf("want 1 request, got %d", count.Load())
	}
}

func TestFetchSendsMethodAndHeaders(t *testing.T) {
	fakeTime(t)
	srv, _ := flaky(t, 0, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds") != "21600" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ok(w, r)
	})
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	hdr := http.Header{}
	hdr.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")
	if body, err := fetch(ctx, http.MethodPut, srv.URL, hdr); err != nil || string(body) != "ok" {
		t.Fatalf("fetch: %q, %v", body, err)
	}
}
