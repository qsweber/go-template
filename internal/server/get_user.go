package server

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/qsweber/go-template/internal/auth"
	"github.com/qsweber/go-template/internal/repositories"
)

// userOutput is the public-facing representation of a user.
type userOutput struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// Email is owned by Cognito rather than duplicated onto the user record, so it comes from
// the caller's verified ID token (callers can only access their own user).
func newUserOutput(user *repositories.User, claims *auth.Claims) userOutput {
	return userOutput{ID: user.ID, Email: claims.Email, DisplayName: user.DisplayName}
}

// authorizeUserRequest authenticates the caller and checks that the requested user id
// (the "id" query parameter) is their own. User ids are the Cognito sub. On failure it
// returns a non-nil error Response.
func (s *ServerImpl) authorizeUserRequest(ctx context.Context, req Request) (string, *auth.Claims, *Response) {
	claims, err := s.authenticateRequest(ctx, req)
	if err != nil {
		resp := errorResponse(401, err)
		return "", nil, &resp
	}

	id := req.QueryParams["id"]
	if id == "" {
		resp := errorResponse(400, errors.New("id query parameter is required"))
		return "", nil, &resp
	}

	if id != claims.Subject {
		resp := errorResponse(403, errors.New("cannot access another user's record"))
		return "", nil, &resp
	}

	return id, claims, nil
}

func (s *ServerImpl) GetUser(ctx context.Context, req Request) Response {
	id, claims, errResp := s.authorizeUserRequest(ctx, req)
	if errResp != nil {
		return *errResp
	}

	user, err := s.serviceContext.Repositories.Users.GetUser(ctx, id)
	if err != nil {
		return errorResponse(500, err)
	}
	if user == nil {
		return errorResponse(404, errors.New("user not found"))
	}

	body, err := json.Marshal(newUserOutput(user, claims))
	if err != nil {
		return errorResponse(500, err)
	}

	return Response{StatusCode: 200, Body: string(body), Headers: corsHeaders()}
}
