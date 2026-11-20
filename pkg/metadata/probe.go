package main

import (
	"context"
	"errors"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ProviderEC2 resolves the endpoint shared by AWS and OpenStack.
type ProviderEC2 struct{ found Provider }

// NewEC2 returns a new shared EC2-style probe.
func NewEC2() *ProviderEC2 { return &ProviderEC2{} }

func (p *ProviderEC2) String() string {
	if p.found != nil {
		return p.found.String()
	}
	return "aws/openstack"
}

// Probe checks the EC2 hostname before distinguishing OpenStack from AWS.
func (p *ProviderEC2) Probe(ctx context.Context) bool {
	p.found = nil
	if _, err := awsGet(ctx, ec2Base+"latest/meta-data/hostname"); err != nil {
		return false
	}
	_, err := get(ctx, ec2Base+"openstack/latest/meta_data.json", nil)
	var se *statusError
	switch {
	case err == nil:
		p.found = NewOpenstack()
	case errors.As(err, &se):
		p.found = NewAWS()
	default:
		return false
	}
	return true
}

// Extract delegates to the provider selected by Probe.
func (p *ProviderEC2) Extract(ctx context.Context) ([]byte, error) {
	if p.found == nil {
		return nil, errors.New("EC2 metadata provider has not been probed")
	}
	return p.found.Extract(ctx)
}

// probeParallel prefers local data, then the first successful network rank.
func probeParallel(ctx context.Context, ranked []Provider, local Provider) Provider {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	n := len(ranked)
	if local != nil {
		n++
	}
	if n == 0 {
		return nil
	}
	type result struct {
		index int
		ok    bool
	}
	// Every probe can send once even after selection stops receiving.
	results := make(chan result, n)
	probe := func(index int, p Provider) {
		results <- result{index, p.Probe(ctx)}
	}
	for i, p := range ranked {
		go probe(i, p)
	}
	if local != nil {
		go probe(len(ranked), local)
	}

	answered := make([]bool, n)
	done := make([]bool, len(ranked))
	next, winner := 0, -1
	localDone := local == nil
	for winner < 0 {
		r := <-results
		answered[r.index] = r.ok
		if r.index == len(ranked) {
			localDone = true
			if r.ok {
				winner = r.index
				break
			}
		} else {
			done[r.index] = true
		}
		for next < len(ranked) && done[next] && !answered[next] {
			next++
		}
		if localDone {
			if next == len(ranked) {
				return nil
			}
			if answered[next] {
				winner = next
			}
		}
	}
	cancel()

	// Include successes already queued, without waiting for cancelled probes.
drain:
	for {
		select {
		case r := <-results:
			answered[r.index] = r.ok
		default:
			break drain
		}
	}
	used := local
	if winner < len(ranked) {
		used = ranked[winner]
	}
	var others []string
	for i, ok := range answered {
		if !ok || i == winner {
			continue
		}
		if i < len(ranked) {
			others = append(others, ranked[i].String())
		} else {
			others = append(others, local.String())
		}
	}
	if len(others) != 0 {
		log.Errorf("metadata: multiple providers answered: %s (used), %s", used, strings.Join(others, ", "))
	}
	return used
}
