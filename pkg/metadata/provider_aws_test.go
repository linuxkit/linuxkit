package main

import (
	"net/http"
	"net/http/httptest"
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

// fakeIMDS serves the token endpoint at /latest/api/token and "ok" elsewhere.
// tokenStatus != 200 makes the token PUT fail with that status; requireToken
// makes GETs without the token fail with 401. The first dropPuts PUTs and
// dropGets GETs are dropped without a response. Returned counts include drops.
func fakeIMDS(t *testing.T, tokenStatus int, requireToken bool, dropPuts, dropGets int) (*httptest.Server, *int, *int) {
	puts, gets := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			if r.Method != http.MethodPut || r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds") != "21600" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			puts++
			if puts <= dropPuts {
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
		gets++
		if gets <= dropGets {
			drop(t, w)
			return
		}
		if requireToken && r.Header.Get("X-aws-ec2-metadata-token") != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("ok"))
	}))
	// fresh connection per request, so net/http never transparently retries
	srv.Config.SetKeepAlivesEnabled(false)
	awsTokenURL = srv.URL + "/latest/api/token"
	awsTokenFetched, awsToken, awsTokenErr = false, "", nil
	delays := awsRetryDelays
	awsRetryDelays = []time.Duration{time.Millisecond, time.Millisecond}
	t.Cleanup(func() { awsRetryDelays = delays; srv.Close() })
	return srv, &puts, &gets
}

func TestAWSGetIMDSv2(t *testing.T) {
	srv, puts, gets := fakeIMDS(t, 200, true, 0, 0)
	for i := 0; i < 3; i++ {
		body, err := awsGet(srv.URL + "/latest/meta-data/hostname")
		if err != nil || string(body) != "ok" {
			t.Fatalf("awsGet: %q, %v", body, err)
		}
	}
	if *puts != 1 || *gets != 3 {
		t.Fatalf("want 1 PUT and 3 GETs, got %d and %d", *puts, *gets)
	}
}

func TestAWSGetIMDSv1Fallback(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusMethodNotAllowed} {
		srv, puts, _ := fakeIMDS(t, status, false, 0, 0)
		body, err := awsGet(srv.URL + "/latest/meta-data/hostname")
		if err != nil || string(body) != "ok" {
			t.Fatalf("status %d: awsGet: %q, %v", status, body, err)
		}
		if *puts != 1 {
			t.Fatalf("status %d: want 1 PUT, got %d", status, *puts)
		}
	}
}

func TestAWSGetUnreachable(t *testing.T) {
	srv, _, gets := fakeIMDS(t, 200, false, 0, 0)
	dead := httptest.NewServer(http.NotFoundHandler())
	dead.Close()
	awsTokenURL = dead.URL + "/latest/api/token"
	if _, err := awsGet(srv.URL + "/latest/meta-data/hostname"); err == nil {
		t.Fatal("awsGet succeeded with unreachable token endpoint")
	}
	if *gets != 0 {
		t.Fatalf("want no GETs, got %d", *gets)
	}
}

func TestAWSTokenRetry(t *testing.T) {
	srv, puts, gets := fakeIMDS(t, 200, true, 1, 0)
	body, err := awsGet(srv.URL + "/latest/meta-data/hostname")
	if err != nil || string(body) != "ok" {
		t.Fatalf("awsGet: %q, %v", body, err)
	}
	if *puts != 2 || *gets != 1 {
		t.Fatalf("want 2 PUTs and 1 GET, got %d and %d", *puts, *gets)
	}
}

func TestAWSTokenRetryExhausted(t *testing.T) {
	srv, puts, gets := fakeIMDS(t, 200, false, 3, 0)
	if _, err := awsGet(srv.URL + "/latest/meta-data/hostname"); err == nil {
		t.Fatal("awsGet succeeded with all token PUTs dropped")
	}
	if *puts != 3 || *gets != 0 {
		t.Fatalf("want 3 PUTs and no GETs, got %d and %d", *puts, *gets)
	}
}

func TestAWSGetRetry(t *testing.T) {
	srv, _, gets := fakeIMDS(t, 200, true, 0, 1)
	body, err := awsGet(srv.URL + "/latest/meta-data/hostname")
	if err != nil || string(body) != "ok" {
		t.Fatalf("awsGet: %q, %v", body, err)
	}
	if *gets != 2 {
		t.Fatalf("want 2 GETs, got %d", *gets)
	}
}

func TestAWSGetNotFoundNoRetry(t *testing.T) {
	srv, _, gets := fakeIMDS(t, 200, true, 0, 0)
	if _, err := awsGet(srv.URL + "/missing"); err == nil {
		t.Fatal("awsGet succeeded on 404")
	}
	if *gets != 1 {
		t.Fatalf("want 1 GET, got %d", *gets)
	}
}
