package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
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
	// fake clock: only the recorded sleeps advance it
	now, sleep := awsNow, awsSleep
	clock := time.Unix(0, 0)
	sleeps = nil
	awsNow = func() time.Time { return clock }
	awsSleep = func(d time.Duration) { sleeps = append(sleeps, d); clock = clock.Add(d) }
	t.Cleanup(func() { awsNow, awsSleep = now, sleep; srv.Close() })
	return srv, &puts, &gets
}

// sleeps records the retry sleeps since the last fakeIMDS
var sleeps []time.Duration

func checkSleeps(t *testing.T, want ...time.Duration) {
	t.Helper()
	if !reflect.DeepEqual(sleeps, want) {
		t.Fatalf("want sleeps %v, got %v", want, sleeps)
	}
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
		checkSleeps(t)
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
	srv, puts, gets := fakeIMDS(t, 200, true, 3, 0)
	body, err := awsGet(srv.URL + "/latest/meta-data/hostname")
	if err != nil || string(body) != "ok" {
		t.Fatalf("awsGet: %q, %v", body, err)
	}
	if *puts != 4 || *gets != 1 {
		t.Fatalf("want 4 PUTs and 1 GET, got %d and %d", *puts, *gets)
	}
	checkSleeps(t, time.Second, 2*time.Second, 4*time.Second)
}

func TestAWSTokenRetryDeadline(t *testing.T) {
	srv, puts, gets := fakeIMDS(t, 200, false, 1000, 0)
	if _, err := awsGet(srv.URL + "/latest/meta-data/hostname"); err == nil {
		t.Fatal("awsGet succeeded with all token PUTs dropped")
	}
	if *puts != 9 || *gets != 0 {
		t.Fatalf("want 9 PUTs and no GETs, got %d and %d", *puts, *gets)
	}
	s := time.Second
	// 1+2+4+5*4 = 27s, then the last sleep is clamped to the 3s left
	checkSleeps(t, s, 2*s, 4*s, 5*s, 5*s, 5*s, 5*s, 3*s)
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
	checkSleeps(t, time.Second)
}

func TestAWSGetNotFoundNoRetry(t *testing.T) {
	srv, _, gets := fakeIMDS(t, 200, true, 0, 0)
	if _, err := awsGet(srv.URL + "/missing"); err == nil {
		t.Fatal("awsGet succeeded on 404")
	}
	if *gets != 1 {
		t.Fatalf("want 1 GET, got %d", *gets)
	}
	checkSleeps(t)
}
