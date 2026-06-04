package server

import (
	"context"
	"encoding/json"
)

type recordClickOutput struct {
	Ok bool `json:"ok"`
}

func (s *ServerImpl) RecordClick(ctx context.Context, req Request) Response {
	claims, err := s.authenticateRequest(ctx, req)
	if err != nil {
		return errorResponse(401, err)
	}

	if err := s.serviceContext.Repositories.Clicks.RecordClick(ctx, claims.CognitoUser); err != nil {
		return errorResponse(500, err)
	}

	body, err := json.Marshal(recordClickOutput{Ok: true})
	if err != nil {
		return errorResponse(500, err)
	}

	return Response{StatusCode: 200, Body: string(body), Headers: corsHeaders()}
}
