package rpc

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/qsweber/go-template/internal/auth"
	"github.com/qsweber/go-template/internal/server"
)

type Request struct {
	Path    string
	Headers map[string]any
}

type Response struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

type TokenVerifier interface {
	VerifyToken(ctx context.Context, tokenString string) (*auth.Claims, error)
}

func authenticateRequest(req Request, tokenVerifier TokenVerifier) (*auth.Claims, error) {
	if tokenVerifier == nil {
		return nil, errors.New("Authentication is not configured")
	}
	authHeader, ok := req.Headers["Authorization"].(string)
	if !ok {
		authHeader, ok = req.Headers["authorization"].(string)
		if !ok {
			return nil, errors.New("Authorization header is missing")
		}
	}
	token, err := auth.ExtractBearerToken(authHeader)
	if err != nil {
		return nil, errors.New("Failed to extract bearer token")
	}
	claims, err := tokenVerifier.VerifyToken(context.Background(), token)
	if err != nil {
		return nil, errors.New("Failed to verify token")
	}
	return claims, nil
}

func corsHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin": "*",
	}
}

func Handler(req Request, srv server.Server, tokenVerifier TokenVerifier) Response {
	switch req.Path {
	case "/ping":
		result, err := srv.Ping()
		if err != nil {
			return Response{StatusCode: 500, Headers: corsHeaders()}
		}
		if result.Ok {
			return Response{StatusCode: 200, Body: `{"ok": true}`, Headers: corsHeaders()}
		}
		return Response{StatusCode: 500, Headers: corsHeaders()}
	case "/foo":
		_, err := authenticateRequest(req, tokenVerifier)
		if err != nil {
			return Response{StatusCode: 401, Headers: corsHeaders()}
		}
		output, err := srv.Foo(server.FooInput{Bar: "example"})
		if err != nil {
			return Response{StatusCode: 500, Headers: corsHeaders()}
		}
		body, err := json.Marshal(output)
		if err != nil {
			return Response{StatusCode: 500, Headers: corsHeaders()}
		}
		return Response{StatusCode: 200, Body: string(body), Headers: corsHeaders()}

	default:
		return Response{StatusCode: 404, Headers: corsHeaders()}
	}
}
