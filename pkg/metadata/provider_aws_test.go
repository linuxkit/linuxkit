package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeIMDS serves the token endpoint at /latest/api/token and "ok" elsewhere.
// tokenStatus != 200 makes the token PUT fail with that status; requireToken
// makes GETs without the token fail with 401.
func fakeIMDS(t *testing.T, tokenStatus int, requireToken bool) (*httptest.Server, *int, *int) {
	puts, gets := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			if r.Method != http.MethodPut || r.Header.Get("X-aws-ec2-metadata-token-ttl-seconds") != "21600" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			puts++
			if tokenStatus != 200 {
				w.WriteHeader(tokenStatus)
				return
			}
			w.Write([]byte("tok"))
			return
		}
		gets++
		if requireToken && r.Header.Get("X-aws-ec2-metadata-token") != "tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte("ok"))
	}))
	awsTokenURL = srv.URL + "/latest/api/token"
	awsTokenFetched, awsToken, awsTokenErr = false, "", nil
	t.Cleanup(srv.Close)
	return srv, &puts, &gets
}

func TestAWSGetIMDSv2(t *testing.T) {
	srv, puts, gets := fakeIMDS(t, 200, true)
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
		srv, puts, _ := fakeIMDS(t, status, false)
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
	srv, _, gets := fakeIMDS(t, 200, false)
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
