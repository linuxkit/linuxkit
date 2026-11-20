package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// fakeIMDS serves the token endpoint at /latest/api/token and "ok" elsewhere,
// and points ec2Base at it. tokenStatus != 200 makes the token PUT fail with
// that status; requireToken makes GETs without the token fail with 401. The
// first dropPuts PUTs are dropped without a response. Counts include drops.
func fakeIMDS(t *testing.T, tokenStatus int, requireToken bool, dropPuts int32) (context.Context, *atomic.Int32, *atomic.Int32) {
	var puts, gets atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			if r.Method != http.MethodPut || r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds") != "21600" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if puts.Add(1) <= dropPuts {
				drop(t, w)
				return
			}
			if tokenStatus != 200 {
				w.WriteHeader(tokenStatus)
				return
			}
			w.Write([]byte("tok"))
			return
		}
		gets.Add(1)
		if requireToken && r.Header.Get("X-aws-ec2-metadata-token") != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ok(w, r)
	}))
	// fresh connection per request, so net/http never transparently retries
	srv.Config.SetKeepAlivesEnabled(false)
	fakeTime(t)
	base := ec2Base
	ec2Base = srv.URL + "/"
	ctx, cancel := withBudget(context.Background())
	t.Cleanup(func() {
		cancel()
		srv.Close()
		ec2Base = base
		awsTokenFetched, awsToken = false, ""
	})
	return ctx, &puts, &gets
}

func TestAWSGetIMDSv2(t *testing.T) {
	ctx, puts, gets := fakeIMDS(t, 200, true, 0)
	for i := 0; i < 3; i++ {
		body, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
		if err != nil || string(body) != "ok" {
			t.Fatalf("awsGet: %q, %v", body, err)
		}
	}
	if puts.Load() != 1 || gets.Load() != 3 {
		t.Fatalf("want 1 PUT and 3 GETs, got %d and %d", puts.Load(), gets.Load())
	}
}

func TestAWSGetIMDSv1Fallback(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusMethodNotAllowed} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			ctx, puts, _ := fakeIMDS(t, status, false, 0)
			body, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
			if err != nil || string(body) != "ok" {
				t.Fatalf("awsGet: %q, %v", body, err)
			}
			if puts.Load() != 1 {
				t.Fatalf("want 1 PUT, got %d", puts.Load())
			}
			checkSleeps(t)
		})
	}
}

func TestAWSTokenRetry(t *testing.T) {
	ctx, puts, gets := fakeIMDS(t, 200, true, 2)
	body, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
	if err != nil || string(body) != "ok" {
		t.Fatalf("awsGet: %q, %v", body, err)
	}
	if puts.Load() != 3 || gets.Load() != 1 {
		t.Fatalf("want 3 PUTs and 1 GET, got %d and %d", puts.Load(), gets.Load())
	}
	checkSleeps(t, time.Second, 2*time.Second)
}

// A token request that never got an answer is not cached: the next budget tries again.
func TestAWSTokenFailureNotCached(t *testing.T) {
	// 9 drops use up the whole budget (see TestGetDeadline)
	ctx, puts, gets := fakeIMDS(t, 200, true, 9)
	if _, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname"); err == nil {
		t.Fatal("awsGet succeeded with all token PUTs dropped")
	}
	if puts.Load() != 9 || gets.Load() != 0 {
		t.Fatalf("want 9 PUTs and no GETs, got %d and %d", puts.Load(), gets.Load())
	}
	ctx, cancel := withBudget(context.Background())
	defer cancel()
	body, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
	if err != nil || string(body) != "ok" {
		t.Fatalf("awsGet: %q, %v", body, err)
	}
	if puts.Load() != 10 {
		t.Fatalf("want 10 PUTs, got %d", puts.Load())
	}
}

func TestAWSGetNotFoundNoRetry(t *testing.T) {
	ctx, _, gets := fakeIMDS(t, 200, true, 0)
	if _, err := awsGet(ctx, ec2Base+"missing"); err == nil {
		t.Fatal("awsGet succeeded on 404")
	}
	if gets.Load() != 1 {
		t.Fatalf("want 1 GET, got %d", gets.Load())
	}
	checkSleeps(t)
}

func TestAWSTokenConcurrent(t *testing.T) {
	ctx := parallelBudget(t)
	gate, release := context.WithCancel(context.Background())
	defer release()
	started := make(chan struct{})
	var puts atomic.Int32
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			if puts.Add(1) == 1 {
				close(started)
			}
			select {
			case <-gate.Done():
			case <-r.Context().Done():
				return
			}
			w.Write([]byte("tok"))
			return
		}
		if r.Header.Get("X-aws-ec2-metadata-token") != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ok(w, r)
	})
	const callers = 16
	results := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() {
			body, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
			if err == nil && string(body) != "ok" {
				err = errors.New("wrong metadata body")
			}
			results <- err
		}()
	}
	<-started
	time.Sleep(50 * time.Millisecond)
	release()
	for i := 0; i < callers; i++ {
		if err := <-results; err != nil {
			t.Errorf("concurrent awsGet: %v", err)
		}
	}
	if puts.Load() != 1 {
		t.Fatalf("want one shared token PUT, got %d", puts.Load())
	}
}

func TestAWSTokenWaiterCancelled(t *testing.T) {
	ctx := parallelBudget(t)
	gate, release := context.WithCancel(context.Background())
	defer release()
	started := make(chan struct{})
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			close(started)
			select {
			case <-gate.Done():
			case <-r.Context().Done():
				return
			}
		}
		ok(w, r)
	})
	producer := make(chan error, 1)
	go func() {
		_, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
		producer <- err
	}()
	<-started
	waitCtx, cancel := context.WithCancel(ctx)
	waiter := make(chan error, 1)
	go func() {
		_, err := awsGet(waitCtx, ec2Base+"latest/meta-data/hostname")
		waiter <- err
	}()
	cancel()
	select {
	case err := <-waiter:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("cancelled token waiter: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("cancelled token waiter blocked behind token request")
	}
	release()
	if err := <-producer; err != nil {
		t.Fatalf("token producer: %v", err)
	}
}

func TestAWSTokenProducerCancelled(t *testing.T) {
	ctx := parallelBudget(t)
	started := make(chan struct{})
	var puts atomic.Int32
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			if puts.Add(1) == 1 {
				close(started)
				<-r.Context().Done()
				return
			}
			w.Write([]byte("tok"))
			return
		}
		if r.Header.Get("X-aws-ec2-metadata-token") != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ok(w, r)
	})
	producerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	producer := make(chan error, 1)
	go func() {
		_, err := awsGet(producerCtx, ec2Base+"latest/meta-data/hostname")
		producer <- err
	}()
	<-started
	waiter := make(chan error, 1)
	go func() {
		_, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname")
		waiter <- err
	}()
	cancel()
	if err := <-producer; !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled producer: %v", err)
	}
	if err := <-waiter; err != nil {
		t.Fatalf("waiter did not retry after cancelled producer: %v", err)
	}
	if puts.Load() != 2 {
		t.Fatalf("want a fresh token request after cancellation, got %d PUTs", puts.Load())
	}
}
