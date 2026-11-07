package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/qsweber/go-template/internal/server"
)

// fakeServer records the request it receives and returns a fixed response.
type fakeServer struct {
	got  server.Request
	resp server.Response
}

func (f *fakeServer) Handle(_ context.Context, req server.Request) server.Response {
	f.got = req
	return f.resp
}

func (f *fakeServer) Ping() server.Response { return server.Response{} }

func (f *fakeServer) GetUser(context.Context, server.Request) server.Response {
	return server.Response{}
}

func (f *fakeServer) UpdateUser(context.Context, server.Request) server.Response {
	return server.Response{}
}

var discardLogger = slog.New(slog.NewJSONHandler(io.Discard, nil))

var putContext = events.APIGatewayV2HTTPRequestContext{
	HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{Method: "PUT"},
}

func TestHandler_MapsRequestAndResponse(t *testing.T) {
	srv := &fakeServer{resp: server.Response{StatusCode: 201, Body: `{"ok":true}`, Headers: map[string]string{"X-Test": "1"}}}

	resp, err := newHandler(srv, discardLogger)(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext:        putContext,
		RawPath:               "/v1/user",
		Headers:               map[string]string{"authorization": "Bearer t"},
		QueryStringParameters: map[string]string{"id": "user-1"},
		Body:                  `{"display_name":"x"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := srv.got
	if got.Method != "PUT" || got.Path != "/v1/user" || got.Body != `{"display_name":"x"}` {
		t.Errorf("request = %+v", got)
	}
	if got.Headers["authorization"] != "Bearer t" || got.QueryParams["id"] != "user-1" {
		t.Errorf("headers/query not passed through: %+v", got)
	}
	if resp.StatusCode != 201 || resp.Body != `{"ok":true}` || resp.Headers["X-Test"] != "1" {
		t.Errorf("response = %+v", resp)
	}
}

func TestHandler_DecodesBase64Body(t *testing.T) {
	srv := &fakeServer{resp: server.Response{StatusCode: 200}}

	_, err := newHandler(srv, discardLogger)(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext:  putContext,
		RawPath:         "/v1/user",
		Body:            base64.StdEncoding.EncodeToString([]byte(`{"display_name":"x"}`)),
		IsBase64Encoded: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv.got.Body != `{"display_name":"x"}` {
		t.Errorf("body = %q, want the decoded JSON", srv.got.Body)
	}
}

func TestHandler_InvalidBase64Body(t *testing.T) {
	srv := &fakeServer{}

	resp, err := newHandler(srv, discardLogger)(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext:  putContext,
		RawPath:         "/v1/user",
		Body:            "not base64!",
		IsBase64Encoded: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if srv.got.Path != "" {
		t.Error("server should not be called with an undecodable body")
	}
}

func TestHandler_LogsOneLinePerRequest(t *testing.T) {
	srv := &fakeServer{resp: server.Response{StatusCode: 404}}
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	_, err := newHandler(srv, logger)(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "req-1",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method:    "GET",
				SourceIP:  "203.0.113.7",
				UserAgent: "scanner/1.0",
			},
		},
		RawPath:               "/.env",
		Headers:               map[string]string{"authorization": "Bearer secret-token"},
		QueryStringParameters: map[string]string{"id": "user-1"},
		Body:                  "secret-body",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want 1: %q", len(lines), buf.String())
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v", err)
	}
	want := map[string]any{
		"msg":       "request",
		"requestId": "req-1",
		"method":    "GET",
		"path":      "/.env",
		"status":    float64(404),
		"sourceIp":  "203.0.113.7",
		"userAgent": "scanner/1.0",
	}
	for key, value := range want {
		if entry[key] != value {
			t.Errorf("%s = %v, want %v", key, entry[key], value)
		}
	}
	if _, ok := entry["durationMs"]; !ok {
		t.Error("durationMs is missing")
	}

	for _, secret := range []string{"secret-token", "secret-body", "user-1"} {
		if strings.Contains(lines[0], secret) {
			t.Errorf("log line leaks %q: %s", secret, lines[0])
		}
	}
}
