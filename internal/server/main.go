package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/qsweber/go-template/internal/auth"
)

type Request struct {
	Method      string
	Path        string
	Headers     map[string]string
	QueryParams map[string]string
	Body        string
}

type Response struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

type Server interface {
	Handle(ctx context.Context, req Request) Response
	Ping() Response
	GetUser(ctx context.Context, req Request) Response
	UpdateUser(ctx context.Context, req Request) Response
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
	switch req.Path {
	case "/v1/ping":
		return s.Ping()
	case "/v1/user":
		switch req.Method {
		case "GET":
			return s.GetUser(ctx, req)
		case "PUT":
			return s.UpdateUser(ctx, req)
		default:
			return errorResponse(405, errors.New("method not allowed"))
		}
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

// corsHeaders are set on every response for clients that call the server directly. Behind API
// Gateway, the HTTP API's CORS configuration (cdk/apigateway.go) answers preflights and takes
// precedence over these headers.
func corsHeaders() map[string]string {
	return map[string]string{
		"Access-Control-Allow-Origin": "*",
		"Cache-Control":               "no-store",
	}
}

func errorResponse(statusCode int, err error) Response {
	body, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		return Response{StatusCode: statusCode, Body: `{"error":"failed to marshal error response"}`, Headers: corsHeaders()}
	}

	return Response{StatusCode: statusCode, Body: string(body), Headers: corsHeaders()}
}
