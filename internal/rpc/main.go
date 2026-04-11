package rpc

import (
	"context"
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
		return nil, errors.New("Authorization header is missing")
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

func Handler(req Request, srv server.Server, tokenVerifier TokenVerifier) Response {
	switch req.Path {
	case "/ping":
		result, err := srv.Ping()
		if err != nil {
			return Response{StatusCode: 500}
		}
		if result.Ok {
			return Response{StatusCode: 200, Body: `{"ok": true}`}
		}
		return Response{StatusCode: 500}
	case "/foo":
		_, err := authenticateRequest(req, tokenVerifier)
		if err != nil {
			return Response{StatusCode: 401}
		}
		output, err := srv.Foo(server.FooInput{Bar: "example"})
		if err != nil {
			return Response{StatusCode: 500}
		}
		return Response{StatusCode: 200, Body: output.String()}

	default:
		return Response{StatusCode: 404}
	}
}
