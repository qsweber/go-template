package main

import (
	"context"
	"encoding/base64"
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

func TestHandler_MapsRequestAndResponse(t *testing.T) {
	srv := &fakeServer{resp: server.Response{StatusCode: 201, Body: `{"ok":true}`, Headers: map[string]string{"X-Test": "1"}}}

	resp, err := newHandler(srv)(context.Background(), events.APIGatewayProxyRequest{
		HTTPMethod:            "PUT",
		Path:                  "/v1/user",
		Headers:               map[string]string{"Authorization": "Bearer t"},
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
	if got.Headers["Authorization"] != "Bearer t" || got.QueryParams["id"] != "user-1" {
		t.Errorf("headers/query not passed through: %+v", got)
	}
	if resp.StatusCode != 201 || resp.Body != `{"ok":true}` || resp.Headers["X-Test"] != "1" {
		t.Errorf("response = %+v", resp)
	}
}

func TestHandler_DecodesBase64Body(t *testing.T) {
	srv := &fakeServer{resp: server.Response{StatusCode: 200}}

	_, err := newHandler(srv)(context.Background(), events.APIGatewayProxyRequest{
		HTTPMethod:      "PUT",
		Path:            "/v1/user",
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

	resp, err := newHandler(srv)(context.Background(), events.APIGatewayProxyRequest{
		HTTPMethod:      "PUT",
		Path:            "/v1/user",
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
