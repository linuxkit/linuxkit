package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

type fakeProvider struct {
	name      string
	ok        bool
	release   chan struct{}
	probes    atomic.Int32
	ctxErr    error // ctx.Err() seen by Extract
	started   chan struct{}
	done      chan struct{}
	cancelled atomic.Bool
}

func (p *fakeProvider) String() string { return p.name }

func (p *fakeProvider) Probe(ctx context.Context) bool {
	p.probes.Add(1)
	if p.started != nil {
		close(p.started)
	}
	if p.done != nil {
		defer close(p.done)
	}
	if p.release != nil {
		select {
		case <-p.release:
		case <-ctx.Done():
			p.cancelled.Store(true)
			return false
		}
	}
	return p.ok
}

func (p *fakeProvider) Extract(ctx context.Context) ([]byte, error) {
	p.ctxErr = ctx.Err()
	return nil, p.ctxErr
}

func parallelBudget(t *testing.T) context.Context {
	t.Helper()
	oldBudget, oldBackoff := budget, retryBackoff
	budget, retryBackoff = 2*time.Second, []time.Duration{50 * time.Millisecond}
	ctx, cancel := withBudget(context.Background())
	t.Cleanup(func() {
		cancel()
		budget, retryBackoff = oldBudget, oldBackoff
	})
	return ctx
}

func probeIMDS(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(h)
	srv.Config.SetKeepAlivesEnabled(false)
	base := ec2Base
	ec2Base = srv.URL + "/"
	t.Cleanup(func() {
		srv.Close()
		ec2Base = base
		awsTokenMu.Lock()
		awsTokenFetched, awsToken = false, ""
		awsTokenMu.Unlock()
	})
}

func captureProbeLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	logger := log.StandardLogger()
	oldOutput, oldFormatter, oldLevel := logger.Out, logger.Formatter, logger.GetLevel()
	log.SetOutput(&logs)
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true, DisableColors: true})
	log.SetLevel(log.InfoLevel)
	t.Cleanup(func() {
		log.SetOutput(oldOutput)
		log.SetFormatter(oldFormatter)
		log.SetLevel(oldLevel)
	})
	return &logs
}

func checkProbeConflict(t *testing.T, logs *bytes.Buffer, message string) {
	t.Helper()
	if !strings.Contains(logs.String(), "level=error") || !strings.Contains(logs.String(), message) {
		t.Fatalf("missing error conflict %q in %s", message, logs.String())
	}
}

func asyncProbe(ctx context.Context, ranked []Provider, local Provider) <-chan Provider {
	result := make(chan Provider, 1)
	go func() { result <- probeParallel(ctx, ranked, local) }()
	return result
}

func awaitProbe(t *testing.T, result <-chan Provider) Provider {
	t.Helper()
	select {
	case p := <-result:
		return p
	case <-time.After(3 * time.Second):
		t.Fatal("parallel probe did not return")
		return nil
	}
}

func assertProbeWaiting(t *testing.T, result <-chan Provider) {
	t.Helper()
	select {
	case p := <-result:
		t.Fatalf("probe returned %v before higher-priority candidate finished", p)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestEC2ProbeOpenStack(t *testing.T) {
	ctx := parallelBudget(t)
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/api/token" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		ok(w, r)
	})
	p := NewEC2()
	if p.String() != "aws/openstack" {
		t.Fatalf("unprobed provider = %s", p)
	}
	if !p.Probe(ctx) || p.String() != "openstack" {
		t.Fatalf("want openstack, got %s", p)
	}
}

func TestEC2ProbeAWS(t *testing.T) {
	ctx := parallelBudget(t)
	var checks atomic.Int32
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/api/token":
			w.Write([]byte("tok"))
		case "/openstack/latest/meta_data.json":
			checks.Add(1)
			w.WriteHeader(http.StatusNotFound)
		default:
			if r.Header.Get("X-aws-ec2-metadata-token") != "tok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			ok(w, r)
		}
	})
	p := NewEC2()
	if !p.Probe(ctx) || p.String() != "AWS" || checks.Load() != 1 {
		t.Fatalf("want AWS with one OpenStack check, got %s, %d checks", p, checks.Load())
	}
}

func TestEC2ProbeV1Fallback(t *testing.T) {
	ctx := parallelBudget(t)
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/api/token":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/openstack/latest/meta_data.json":
			w.WriteHeader(http.StatusNotFound)
		default:
			if r.Header.Get("X-aws-ec2-metadata-token") != "" {
				t.Error("IMDSv1 request included a token")
			}
			ok(w, r)
		}
	})
	p := NewEC2()
	if !p.Probe(ctx) || p.String() != "AWS" {
		t.Fatalf("want AWS, got %s", p)
	}
}

func TestEC2ProbeHostnameRequired(t *testing.T) {
	ctx := parallelBudget(t)
	var checks atomic.Int32
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openstack/latest/meta_data.json" {
			checks.Add(1)
		}
		w.WriteHeader(http.StatusNotFound)
	})
	p := NewEC2()
	if p.Probe(ctx) || checks.Load() != 0 {
		t.Fatalf("failed hostname still selected %s or checked OpenStack %d times", p, checks.Load())
	}
}

func TestEC2ProbeOpenStackRetry(t *testing.T) {
	ctx := parallelBudget(t)
	var checks atomic.Int32
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openstack/latest/meta_data.json" && checks.Add(1) <= 2 {
			drop(t, w)
			return
		}
		ok(w, r)
	})
	p := NewEC2()
	if !p.Probe(ctx) || p.String() != "openstack" || checks.Load() != 3 {
		t.Fatalf("want openstack after retries, got %s, %d checks", p, checks.Load())
	}
}

func TestEC2ProbeOpenStackDeadline(t *testing.T) {
	ctx := parallelBudget(t)
	var checks atomic.Int32
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openstack/latest/meta_data.json" {
			checks.Add(1)
			drop(t, w)
			return
		}
		ok(w, r)
	})
	p := NewEC2()
	start := time.Now()
	if p.Probe(ctx) || p.String() != "aws/openstack" {
		t.Fatalf("unanswered OpenStack check incorrectly selected %s", p)
	}
	if time.Since(start) < budget-100*time.Millisecond || checks.Load() < 2 {
		t.Fatalf("OpenStack check gave up early after %s, %d checks", time.Since(start), checks.Load())
	}
}

func TestEC2ExtractDelegates(t *testing.T) {
	p := NewEC2()
	if _, err := p.Extract(context.Background()); err == nil {
		t.Fatal("unprobed EC2 provider extracted data")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	found := &fakeProvider{name: "found"}
	p.found = found
	if _, err := p.Extract(ctx); !errors.Is(err, context.Canceled) || found.ctxErr != context.Canceled {
		t.Fatalf("Extract did not delegate context: %v, %v", err, found.ctxErr)
	}
}

func TestProbeParallelSingleWinner(t *testing.T) {
	ctx := parallelBudget(t)
	winner := &fakeProvider{name: "f1", ok: true}
	ranked := []Provider{&fakeProvider{name: "f0"}, winner, &fakeProvider{name: "f2"}}
	if got := probeParallel(ctx, ranked, nil); got != winner {
		t.Fatalf("want f1, got %v", got)
	}
}

func TestProbeParallelRankWaitsForHigher(t *testing.T) {
	ctx := parallelBudget(t)
	higher := &fakeProvider{name: "f0", release: make(chan struct{})}
	winner := &fakeProvider{name: "f1", ok: true}
	result := asyncProbe(ctx, []Provider{higher, winner}, nil)
	assertProbeWaiting(t, result)
	close(higher.release)
	if got := awaitProbe(t, result); got != winner {
		t.Fatalf("want f1, got %v", got)
	}
}

func TestProbeParallelTwoAnswer(t *testing.T) {
	ctx := parallelBudget(t)
	logs := captureProbeLogs(t)
	release, answered := make(chan struct{}), make(chan struct{})
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/meta-data/hostname":
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		case "/openstack/latest/meta_data.json":
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ok(w, r)
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok(w, r)
		close(answered)
	}))
	base := vultrBase
	vultrBase = srv.URL + "/"
	t.Cleanup(func() { srv.Close(); vultrBase = base })
	result := asyncProbe(ctx, []Provider{NewEC2(), NewVultr()}, nil)
	<-answered
	assertProbeWaiting(t, result)
	close(release)
	if got := awaitProbe(t, result); got == nil || got.String() != "AWS" {
		t.Fatalf("want AWS, got %v", got)
	}
	checkProbeConflict(t, logs, "metadata: multiple providers answered: AWS (used), Vultr")
}

func TestProbeParallelAllFail(t *testing.T) {
	ctx := parallelBudget(t)
	probeIMDS(t, http.NotFound)
	srv := httptest.NewServer(http.HandlerFunc(http.NotFound))
	base := vultrBase
	vultrBase = srv.URL + "/"
	t.Cleanup(func() { srv.Close(); vultrBase = base })
	start := time.Now()
	if got := probeParallel(ctx, []Provider{NewEC2(), NewVultr()}, &fakeProvider{name: "VMWARE"}); got != nil {
		t.Fatalf("want no provider, got %v", got)
	}
	if elapsed := time.Since(start); elapsed >= budget/2 {
		t.Fatalf("HTTP failures took %s", elapsed)
	}
}

func TestProbeParallelSilentThenAnswers(t *testing.T) {
	ctx := parallelBudget(t)
	start := time.Now()
	var attempts atomic.Int32
	probeIMDS(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		if time.Since(start) < 600*time.Millisecond {
			drop(t, w)
			return
		}
		if r.URL.Path == "/openstack/latest/meta_data.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		ok(w, r)
	})
	srv := httptest.NewServer(http.HandlerFunc(http.NotFound))
	base := vultrBase
	vultrBase = srv.URL + "/"
	t.Cleanup(func() { srv.Close(); vultrBase = base })
	if got := probeParallel(ctx, []Provider{NewEC2(), NewVultr()}, nil); got == nil || got.String() != "AWS" {
		t.Fatalf("want AWS, got %v", got)
	}
	if attempts.Load() < 3 {
		t.Fatalf("want at least three attempts, got %d", attempts.Load())
	}
}

func TestProbeParallelSilentUntilDeadline(t *testing.T) {
	ctx := parallelBudget(t)
	var attempts atomic.Int32
	probeIMDS(t, func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		drop(t, w)
	})
	start := time.Now()
	if got := probeParallel(ctx, []Provider{NewEC2()}, nil); got != nil {
		t.Fatalf("want no provider, got %v", got)
	}
	if elapsed := time.Since(start); elapsed < budget-100*time.Millisecond || elapsed > budget+time.Second {
		t.Fatalf("deadline took %s", elapsed)
	}
	if attempts.Load() <= 1 {
		t.Fatalf("silent service only tried %d times", attempts.Load())
	}
}

func TestProbeParallelVMwareWinsImmediately(t *testing.T) {
	ctx := parallelBudget(t)
	logs := captureProbeLogs(t)
	network := &fakeProvider{name: "network", release: make(chan struct{}), done: make(chan struct{})}
	local := &fakeProvider{name: "VMWARE", ok: true}
	start := time.Now()
	if got := probeParallel(ctx, []Provider{network}, local); got != local {
		t.Fatalf("want VMWARE, got %v", got)
	}
	if elapsed := time.Since(start); elapsed >= 100*time.Millisecond {
		t.Fatalf("guestinfo winner took %s", elapsed)
	}
	select {
	case <-network.done:
	case <-time.After(time.Second):
		t.Fatal("network probe did not finish on cancellation")
	}
	if !network.cancelled.Load() {
		t.Fatal("network probe did not see cancellation")
	}
	if logs.Len() != 0 {
		t.Fatalf("unexpected conflict: %s", logs.String())
	}
}

func TestProbeParallelVMwareConflict(t *testing.T) {
	ctx := parallelBudget(t)
	logs := captureProbeLogs(t)
	local := &fakeProvider{name: "VMWARE", ok: true, release: make(chan struct{})}
	result := asyncProbe(ctx, []Provider{&fakeProvider{name: "f0", ok: true}}, local)
	assertProbeWaiting(t, result)
	close(local.release)
	if got := awaitProbe(t, result); got != local {
		t.Fatalf("want VMWARE, got %v", got)
	}
	checkProbeConflict(t, logs, "metadata: multiple providers answered: VMWARE (used), f0")
}

func TestProbeParallelNetworkWaitsForVMware(t *testing.T) {
	ctx := parallelBudget(t)
	logs := captureProbeLogs(t)
	winner := &fakeProvider{name: "f0", ok: true}
	local := &fakeProvider{name: "VMWARE", release: make(chan struct{})}
	result := asyncProbe(ctx, []Provider{winner}, local)
	assertProbeWaiting(t, result)
	close(local.release)
	if got := awaitProbe(t, result); got != winner {
		t.Fatalf("want f0, got %v", got)
	}
	if logs.Len() != 0 {
		t.Fatalf("unexpected conflict: %s", logs.String())
	}
}

func TestProbeParallelEmpty(t *testing.T) {
	if got := probeParallel(context.Background(), nil, nil); got != nil {
		t.Fatalf("empty candidates selected %v", got)
	}
}

func TestProbeParallelLocalOnly(t *testing.T) {
	local := &fakeProvider{name: "VMWARE", ok: true}
	if got := probeParallel(context.Background(), nil, local); got != local {
		t.Fatalf("want VMWARE, got %v", got)
	}
}

func TestProbeParallelNetworkCancelsLowerRank(t *testing.T) {
	ctx := parallelBudget(t)
	winner := &fakeProvider{name: "f0", ok: true}
	lower := &fakeProvider{name: "f1", release: make(chan struct{}), done: make(chan struct{})}
	if got := probeParallel(ctx, []Provider{winner, lower}, nil); got != winner {
		t.Fatalf("want f0, got %v", got)
	}
	select {
	case <-lower.done:
	case <-time.After(time.Second):
		t.Fatal("lower-ranked probe stranded after winner returned")
	}
	if !lower.cancelled.Load() {
		t.Fatal("lower-ranked probe did not see cancellation")
	}
}
