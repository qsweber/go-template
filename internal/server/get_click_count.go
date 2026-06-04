package server

import (
	"context"
	"encoding/json"
)

type getClickCountOutput struct {
	Count int `json:"count"`
}

func (s *ServerImpl) GetClickCount(ctx context.Context) Response {
	count, err := s.serviceContext.Repositories.Clicks.GetClickCount(ctx)
	if err != nil {
		return errorResponse(500, err)
	}

	body, err := json.Marshal(getClickCountOutput{Count: count})
	if err != nil {
		return errorResponse(500, err)
	}

	return Response{StatusCode: 200, Body: string(body), Headers: corsHeaders()}
}
