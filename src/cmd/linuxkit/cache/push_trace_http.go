package cache

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type uploadRequestTrace struct {
	owner                                                                   *uploadTrace
	id                                                                      uint64
	mu                                                                      sync.Mutex
	method, host, operation, session, digest, connection, protocol          string
	length                                                                  int64
	started, gotConn, wroteHeaders, wroteRequest, firstByte, headers, ended time.Time
	dnsStarted, tlsStarted                                                  time.Time
	dnsTime, connectTime, tlsTime                                           time.Duration
	connectStarted                                                          map[string]time.Time
	connectAttempts, interim, status                                        int
	reused, wasIdle                                                         bool
	idleTime                                                                time.Duration
	writeError, errorKind                                                   string
	responseBytes                                                           int64
	doneOnce                                                                sync.Once
	responseDigest                                                          string
}

func traceSessionKey(u *url.URL) string {
	// Some registries identify sessions entirely in opaque query state. Hash it
	// rather than logging/storing it verbatim. Only the client-added digest is
	// excluded, so the closing PUT still joins the preceding PATCH session.
	query := u.Query()
	query.Del("digest")
	sum := sha256.Sum256([]byte(u.Host + u.EscapedPath() + "?" + query.Encode()))
	return hex.EncodeToString(sum[:])
}
func (t *uploadTrace) sessionLocked(u *url.URL, create bool) string {
	key := traceSessionKey(u)
	if id := t.sessions[key]; id != "" {
		return id
	}
	if !create {
		return ""
	}
	sum := sha256.Sum256([]byte(t.id + "|" + key))
	id := "s-" + hex.EncodeToString(sum[:8])
	t.sessions[key] = id
	return id
}
func requestOperation(req *http.Request, session string) string {
	path := req.URL.Path
	if session != "" || strings.Contains(path, "/blobs/uploads/") {
		switch req.Method {
		case http.MethodPost:
			return "upload_init"
		case http.MethodPatch:
			return "upload_body"
		case http.MethodPut:
			return "upload_commit"
		case http.MethodGet:
			return "upload_status"
		}
	}
	if strings.Contains(path, "/manifests/") {
		switch req.Method {
		case http.MethodHead:
			return "manifest_check"
		case http.MethodPut:
			return "manifest_put"
		case http.MethodGet:
			return "manifest_fetch"
		}
	}
	if strings.Contains(path, "/blobs/") {
		if req.Method == http.MethodHead {
			return "blob_check"
		}
		return "blob_fetch"
	}
	if path == "/v2/" {
		return "registry_ping"
	}
	return "http"
}
func (t *uploadTrace) beginRequest(req *http.Request, publishing v1.Hash) (*http.Request, *uploadRequestTrace) {
	if t == nil {
		return req, nil
	}
	t.mu.Lock()
	t.nextRequest++
	id := t.nextRequest
	create := strings.Contains(req.URL.Path, "/blobs/uploads/") && req.Method != http.MethodPost
	session := t.sessionLocked(req.URL, create)
	digest := ""
	if publishing.Hex != "" {
		digest = publishing.String()
	} else if value := req.URL.Query().Get("digest"); value != "" {
		if hash, err := v1.NewHash(value); err == nil {
			digest = hash.String()
		}
	} else if value := req.URL.Query().Get("mount"); value != "" {
		if hash, err := v1.NewHash(value); err == nil {
			digest = hash.String()
		}
	} else if strings.Contains(req.URL.Path, "/blobs/") || strings.Contains(req.URL.Path, "/manifests/") {
		if hash, err := v1.NewHash(req.URL.Path[strings.LastIndex(req.URL.Path, "/")+1:]); err == nil {
			digest = hash.String()
		}
	}
	if digest == "" {
		digest = t.bindings[session]
	} else if session != "" {
		t.bindings[session] = digest
	}
	r := &uploadRequestTrace{owner: t, id: id, method: req.Method, host: req.URL.Host, operation: requestOperation(req, session), session: session, digest: digest, length: req.ContentLength, started: time.Now(), connectStarted: map[string]time.Time{}}
	t.requests[id] = r
	t.mu.Unlock()
	t.emit(r.snapshot("request_start", time.Now()))
	if session != "" && digest != "" {
		t.emit(map[string]interface{}{"event": "session_bind", "request_id": id, "session_id": session, "digest": digest})
	}
	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { r.mu.Lock(); r.dnsStarted = time.Now(); r.mu.Unlock() },
		DNSDone: func(httptrace.DNSDoneInfo) {
			r.mu.Lock()
			if !r.dnsStarted.IsZero() {
				r.dnsTime += time.Since(r.dnsStarted)
				r.dnsStarted = time.Time{}
			}
			r.mu.Unlock()
		},
		ConnectStart: func(network, address string) {
			r.mu.Lock()
			r.connectAttempts++
			r.connectStarted[network+"|"+address] = time.Now()
			r.mu.Unlock()
		},
		ConnectDone: func(network, address string, err error) {
			r.mu.Lock()
			key := network + "|" + address
			if started, ok := r.connectStarted[key]; ok {
				r.connectTime += time.Since(started)
				delete(r.connectStarted, key)
			}
			r.mu.Unlock()
		},
		TLSHandshakeStart: func() { r.mu.Lock(); r.tlsStarted = time.Now(); r.mu.Unlock() },
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			r.mu.Lock()
			if !r.tlsStarted.IsZero() {
				r.tlsTime += time.Since(r.tlsStarted)
				r.tlsStarted = time.Time{}
			}
			r.mu.Unlock()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			t.mu.Lock()
			connection := t.connections[info.Conn]
			if connection == "" {
				connection = fmt.Sprintf("c-%d", len(t.connections)+1)
				t.connections[info.Conn] = connection
			}
			t.mu.Unlock()
			r.mu.Lock()
			r.gotConn = time.Now()
			r.connection = connection
			r.reused = info.Reused
			r.wasIdle = info.WasIdle
			r.idleTime = info.IdleTime
			r.mu.Unlock()
		},
		WroteHeaders: func() { r.mu.Lock(); r.wroteHeaders = time.Now(); r.mu.Unlock() },
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.mu.Lock()
			r.wroteRequest = time.Now()
			r.writeError = traceErrorKind(info.Err)
			r.mu.Unlock()
			t.emit(r.snapshot("request_body_done", time.Now()))
		},
		GotFirstResponseByte: func() {
			r.mu.Lock()
			if r.firstByte.IsZero() {
				r.firstByte = time.Now()
			}
			r.mu.Unlock()
		},
		Got1xxResponse: func(int, textproto.MIMEHeader) error { r.mu.Lock(); r.interim++; r.mu.Unlock(); return nil },
	}
	return req.Clone(httptrace.WithClientTrace(req.Context(), trace)), r
}
func (r *uploadRequestTrace) snapshot(event string, now time.Time) map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	end := now
	if !r.ended.IsZero() {
		end = r.ended
	}
	phase := "waiting_connection"
	if !r.gotConn.IsZero() {
		phase = "sending"
	}
	if !r.wroteRequest.IsZero() {
		phase = "waiting_headers"
	}
	if !r.headers.IsZero() {
		phase = "reading_response"
	}
	if !r.ended.IsZero() {
		phase = "done"
	}
	record := map[string]interface{}{"event": event, "request_id": r.id, "method": r.method, "host": r.host, "operation": r.operation, "phase": phase, "content_length": r.length, "duration_ms": milliseconds(end.Sub(r.started)), "status": r.status, "http_protocol": r.protocol, "response_bytes": r.responseBytes, "interim_responses": r.interim, "connect_attempts": r.connectAttempts, "error_kind": r.errorKind}
	if r.session != "" {
		record["session_id"] = r.session
	}
	if r.digest != "" {
		record["digest"] = r.digest
	}
	if r.responseDigest != "" {
		record["response_digest"] = r.responseDigest
	}
	if r.connection != "" {
		record["connection_id"] = r.connection
		record["connection_reused"] = r.reused
		record["connection_was_idle"] = r.wasIdle
		record["connection_idle_ms"] = milliseconds(r.idleTime)
	}
	if r.dnsTime > 0 {
		record["dns_ms"] = milliseconds(r.dnsTime)
	}
	if r.connectTime > 0 {
		record["connect_ms"] = milliseconds(r.connectTime)
	}
	if r.tlsTime > 0 {
		record["tls_ms"] = milliseconds(r.tlsTime)
	}
	if !r.gotConn.IsZero() {
		record["connection_wait_ms"] = milliseconds(r.gotConn.Sub(r.started))
	}
	if !r.wroteHeaders.IsZero() {
		record["headers_sent_ms"] = milliseconds(r.wroteHeaders.Sub(r.started))
	}
	if !r.wroteRequest.IsZero() {
		record["request_written_ms"] = milliseconds(r.wroteRequest.Sub(r.started))
		record["write_error_kind"] = r.writeError
		if !r.wroteHeaders.IsZero() && !r.wroteRequest.Before(r.wroteHeaders) {
			record["body_send_ms"] = milliseconds(r.wroteRequest.Sub(r.wroteHeaders))
		}
	}
	if !r.firstByte.IsZero() {
		record["first_response_ms"] = milliseconds(r.firstByte.Sub(r.started))
	}
	if !r.headers.IsZero() {
		record["response_headers_ms"] = milliseconds(r.headers.Sub(r.started))
		if !r.wroteRequest.IsZero() && !r.headers.Before(r.wroteRequest) {
			record["response_wait_ms"] = milliseconds(r.headers.Sub(r.wroteRequest))
		} else if r.length > 0 {
			record["early_response"] = true
		}
		if !r.ended.IsZero() {
			record["response_body_ms"] = milliseconds(r.ended.Sub(r.headers))
		}
	}
	return record
}
func (r *uploadRequestTrace) response(req *http.Request, resp *http.Response, err error) {
	if r == nil {
		return
	}
	if err != nil {
		r.finish(err)
		return
	}
	r.mu.Lock()
	r.headers = time.Now()
	r.status = resp.StatusCode
	r.protocol = resp.Proto
	if hash, err := v1.NewHash(resp.Header.Get("Docker-Content-Digest")); err == nil {
		r.responseDigest = hash.String()
		if r.digest == "" {
			r.digest = r.responseDigest
		}
	}
	session, digest := r.session, r.digest
	r.mu.Unlock()
	if location := resp.Header.Get("Location"); location != "" && (r.operation == "upload_init" || r.operation == "upload_body" || r.operation == "upload_status") {
		if next, parseErr := url.Parse(location); parseErr == nil {
			resolved := req.URL.ResolveReference(next)
			r.owner.mu.Lock()
			if session == "" && r.operation == "upload_init" {
				session = r.owner.sessionLocked(resolved, true)
			} else if session != "" {
				r.owner.sessions[traceSessionKey(resolved)] = session
			}
			if session != "" && digest != "" {
				r.owner.bindings[session] = digest
			}
			r.owner.mu.Unlock()
			r.mu.Lock()
			r.session = session
			r.mu.Unlock()
			if session != "" {
				record := map[string]interface{}{"event": "session_location", "request_id": r.id, "session_id": session}
				if digest != "" {
					record["digest"] = digest
				}
				r.owner.emit(record)
			}
		}
	}
	r.owner.emit(r.snapshot("request_headers", time.Now()))
	if resp.Body == nil || resp.Body == http.NoBody {
		r.finish(nil)
	} else {
		resp.Body = &uploadTraceResponseBody{ReadCloser: resp.Body, request: r}
	}
}
func (r *uploadRequestTrace) finish(err error) {
	r.doneOnce.Do(func() {
		r.mu.Lock()
		r.ended = time.Now()
		r.errorKind = traceErrorKind(err)
		if err == nil && r.status >= 400 {
			r.errorKind = "http_status"
		}
		r.mu.Unlock()
		r.owner.mu.Lock()
		delete(r.owner.requests, r.id)
		r.owner.mu.Unlock()
		r.owner.emit(r.snapshot("request_done", time.Now()))
	})
}

type uploadTraceResponseBody struct {
	io.ReadCloser
	request *uploadRequestTrace
}

func (b *uploadTraceResponseBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.request.mu.Lock()
	b.request.responseBytes += int64(n)
	b.request.mu.Unlock()
	if err != nil {
		if err == io.EOF {
			b.request.finish(nil)
		} else {
			b.request.finish(err)
		}
	}
	return n, err
}
func (b *uploadTraceResponseBody) Close() error {
	err := b.ReadCloser.Close()
	b.request.finish(err)
	return err
}
