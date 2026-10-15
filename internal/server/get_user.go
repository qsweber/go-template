package server

import (
	"context"
	"encoding/json"
	"errors"
)

// userOutput is the public-facing representation of a user.
type userOutput struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (s *ServerImpl) GetUser(ctx context.Context, req Request) Response {
	claims, err := s.authenticateRequest(ctx, req)
	if err != nil {
		return errorResponse(401, err)
	}

	id := req.QueryParams["id"]
	if id == "" {
		return errorResponse(400, errors.New("id query parameter is required"))
	}

	// Users may only fetch their own record. User ids are the Cognito sub.
	if id != claims.Subject {
		return errorResponse(403, errors.New("cannot access another user's record"))
	}

	user, err := s.serviceContext.Repositories.Users.GetUser(ctx, id)
	if err != nil {
		return errorResponse(500, err)
	}
	if user == nil {
		return errorResponse(404, errors.New("user not found"))
	}

	body, err := json.Marshal(userOutput{ID: user.ID, Email: user.Email})
	if err != nil {
		return errorResponse(500, err)
	}

	return Response{StatusCode: 200, Body: string(body), Headers: corsHeaders()}
}
