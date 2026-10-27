package main

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/qsweber/go-template/internal/repositories"
)

type fakeUsers struct {
	createdIDs []string
	err        error
}

func (f *fakeUsers) CreateUser(_ context.Context, id string) error {
	f.createdIDs = append(f.createdIDs, id)
	return f.err
}

func (f *fakeUsers) GetUser(context.Context, string) (*repositories.User, error) {
	return nil, errors.New("not used")
}

func (f *fakeUsers) UpdateDisplayName(context.Context, string, string) (*repositories.User, error) {
	return nil, errors.New("not used")
}

func postConfirmationEvent(triggerSource string, attributes map[string]string) events.CognitoEventUserPoolsPostConfirmation {
	var event events.CognitoEventUserPoolsPostConfirmation
	event.TriggerSource = triggerSource
	event.UserName = "username"
	event.Request.UserAttributes = attributes
	return event
}

func TestHandler_CreatesUserOnSignUp(t *testing.T) {
	users := &fakeUsers{}
	event := postConfirmationEvent("PostConfirmation_ConfirmSignUp", map[string]string{"sub": "user-1", "email": "user@example.com"})

	got, err := newHandler(users)(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users.createdIDs) != 1 || users.createdIDs[0] != "user-1" {
		t.Errorf("created %v, want [user-1]", users.createdIDs)
	}
	// Cognito requires the trigger to return the event it was given.
	if got.UserName != event.UserName || got.TriggerSource != event.TriggerSource {
		t.Errorf("returned event %+v, want the input event", got)
	}
}

func TestHandler_IgnoresForgotPassword(t *testing.T) {
	users := &fakeUsers{}
	event := postConfirmationEvent("PostConfirmation_ConfirmForgotPassword", map[string]string{"sub": "user-1"})

	if _, err := newHandler(users)(context.Background(), event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users.createdIDs) != 0 {
		t.Errorf("created %v, want no users", users.createdIDs)
	}
}

func TestHandler_MissingSub(t *testing.T) {
	users := &fakeUsers{}
	event := postConfirmationEvent("PostConfirmation_ConfirmSignUp", map[string]string{"email": "user@example.com"})

	if _, err := newHandler(users)(context.Background(), event); err == nil {
		t.Error("expected an error when sub is missing")
	}
	if len(users.createdIDs) != 0 {
		t.Errorf("created %v, want no users", users.createdIDs)
	}
}

func TestHandler_RepositoryError(t *testing.T) {
	wantErr := errors.New("boom")
	event := postConfirmationEvent("PostConfirmation_ConfirmSignUp", map[string]string{"sub": "user-1"})

	// Returning the error fails the confirmation, so Cognito surfaces it rather than
	// leaving a confirmed user with no record.
	if _, err := newHandler(&fakeUsers{err: wantErr})(context.Background(), event); !errors.Is(err, wantErr) {
		t.Errorf("error = %v, want %v", err, wantErr)
	}
}
