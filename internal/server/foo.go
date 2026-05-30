package server

import (
	"context"
	"encoding/json"
)

type fooOutput struct {
	Baz string `json:"baz"`
}

func (s *ServerImpl) Foo(ctx context.Context, req Request) Response {
	claims, err := s.authenticateRequest(ctx, req)
	if err != nil {
		return errorResponse(401, err)
	}

	// Record the click for the authenticated user.
	if err := s.serviceContext.Repositories.Clicks.RecordClick(ctx, claims.CognitoUser); err != nil {
		return errorResponse(500, err)
	}

	body, err := json.Marshal(fooOutput{Baz: "example"})
	if err != nil {
		return errorResponse(500, err)
	}

	return Response{StatusCode: 200, Body: string(body), Headers: corsHeaders()}
}
