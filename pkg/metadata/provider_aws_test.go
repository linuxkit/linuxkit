package main

import (
	"context"
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
