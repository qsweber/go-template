package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxDisplayNameLength = 50

type updateUserInput struct {
	DisplayName *string `json:"display_name"`
}

func (s *ServerImpl) UpdateUser(ctx context.Context, req Request) Response {
	id, claims, errResp := s.authorizeUserRequest(ctx, req)
	if errResp != nil {
		return *errResp
	}

	var input updateUserInput
	if err := json.Unmarshal([]byte(req.Body), &input); err != nil {
		return errorResponse(400, fmt.Errorf("invalid request body: %w", err))
	}
	if input.DisplayName == nil {
		return errorResponse(400, errors.New("display_name is required"))
	}

	displayName, err := validateDisplayName(*input.DisplayName)
	if err != nil {
		return errorResponse(400, err)
	}

	user, err := s.serviceContext.Repositories.Users.UpdateDisplayName(ctx, id, displayName)
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

// validateDisplayName trims surrounding whitespace and checks the length and characters.
// An empty result is valid and clears the display name.
func validateDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if utf8.RuneCountInString(displayName) > maxDisplayNameLength {
		return "", fmt.Errorf("display_name must be at most %d characters", maxDisplayNameLength)
	}
	for _, r := range displayName {
		if unicode.IsControl(r) {
			return "", errors.New("display_name must not contain control characters")
		}
	}
	return displayName, nil
}
