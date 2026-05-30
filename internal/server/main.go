package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/qsweber/go-template/internal/auth"
	"github.com/qsweber/go-template/internal/repositories"
)

type Request struct {
	Method  string
	Path    string
	Headers map[string]string
}

type Response struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

type ServiceContext struct {
	Repositories    repositories.Repositories
	CognitoVerifier *auth.CognitoVerifier
}

type Server interface {
	Handle(ctx context.Context, req Request) Response
	Ping() Response
	Foo(ctx context.Context, req Request) Response
}

type ServerImpl struct {
	serviceContext ServiceContext
}

func New(serviceContext ServiceContext) Server {
	return &ServerImpl{
		serviceContext: serviceContext,
	}
}

func (s *ServerImpl) Handle(ctx context.Context, req Request) Response {
	if req.Method == "OPTIONS" {
		return Response{StatusCode: 200, Headers: preflightHeaders()}
	}

	switch req.Path {
	case "/ping":
		return s.Ping()
	case "/foo":
		return s.Foo(ctx, req)
	default:
		return errorResponse(404, errors.New("route not found"))
	}
}

func (s *ServerImpl) authenticateRequest(ctx context.Context, req Request) (*auth.Claims, error) {
	if s.serviceContext.CognitoVerifier == nil {
		return nil, errors.New("authentication is not configured")
	}
	authHeader, ok := req.Headers["Authorization"]
	if !ok {
		authHeader, ok = req.Headers["authorization"]
		if !ok {
			return nil, errors.New("authorization header is missing")
		}
	}
	token, err := auth.ExtractBearerToken(authHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract bearer token: %w", err)
	}
	claims, err := s.serviceContext.CognitoVerifier.VerifyToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	return claims, nil
}

func corsHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin": "*",
	}
}

func preflightHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}
}

func errorResponse(statusCode int, err error) Response {
	body, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		return Response{StatusCode: statusCode, Body: `{"error":"failed to marshal error response"}`, Headers: corsHeaders()}
	}

	return Response{StatusCode: statusCode, Body: string(body), Headers: corsHeaders()}
}
