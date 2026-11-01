package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/qsweber/go-template/internal/auth"
	"github.com/qsweber/go-template/internal/repositories"
)

const (
	testUserID = "user-1"
	testToken  = "valid-token"
)

// fakeVerifier accepts only testToken, as the ID token of testUserID.
type fakeVerifier struct{}

func (fakeVerifier) VerifyToken(_ context.Context, token string) (*auth.Claims, error) {
	if token != testToken {
		return nil, errors.New("bad token")
	}
	return &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: testUserID},
		Email:            "user@example.com",
	}, nil
}

// fakeUsers is an in-memory UsersRepository.
type fakeUsers struct {
	users map[string]*repositories.User
	err   error

	updatedID   string
	updatedName string
}

func newFakeUsers(users ...*repositories.User) *fakeUsers {
	f := &fakeUsers{users: map[string]*repositories.User{}}
	for _, u := range users {
		f.users[u.ID] = u
	}
	return f
}

func (f *fakeUsers) CreateUser(_ context.Context, id string) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.users[id]; !ok {
		f.users[id] = &repositories.User{ID: id}
	}
	return nil
}

func (f *fakeUsers) GetUser(_ context.Context, id string) (*repositories.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.users[id], nil
}

func (f *fakeUsers) UpdateDisplayName(_ context.Context, id string, displayName string) (*repositories.User, error) {
	f.updatedID, f.updatedName = id, displayName
	if f.err != nil {
		return nil, f.err
	}
	user, ok := f.users[id]
	if !ok {
		return nil, nil
	}
	user.DisplayName = displayName
	return user, nil
}

func newTestServer(users *fakeUsers) Server {
	return New(ServiceContext{
		Repositories:    repositories.Repositories{Users: users},
		CognitoVerifier: fakeVerifier{},
	})
}

func authedRequest(method string, id string, body string) Request {
	return Request{
		Method:      method,
		Path:        "/v1/user",
		Headers:     map[string]string{"Authorization": "Bearer " + testToken},
		QueryParams: map[string]string{"id": id},
		Body:        body,
	}
}

func decodeBody(t *testing.T, resp Response) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(resp.Body), &body); err != nil {
		t.Fatalf("response body %q is not JSON: %v", resp.Body, err)
	}
	return body
}

func assertStatus(t *testing.T, resp Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("status = %d, want %d (body: %s)", resp.StatusCode, want, resp.Body)
	}
	if resp.Headers["Access-Control-Allow-Origin"] != "*" {
		t.Errorf("missing CORS header on %d response", resp.StatusCode)
	}
}

func TestHandle_Routing(t *testing.T) {
	srv := newTestServer(newFakeUsers())

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "preflight", method: "OPTIONS", path: "/v1/user", wantStatus: 200},
		{name: "ping", method: "GET", path: "/v1/ping", wantStatus: 200},
		{name: "unknown route", method: "GET", path: "/v1/nope", wantStatus: 404},
		{name: "unsupported method on user", method: "DELETE", path: "/v1/user", wantStatus: 405},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := srv.Handle(context.Background(), Request{Method: tt.method, Path: tt.path})
			assertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestHandle_PreflightAllowsPut(t *testing.T) {
	resp := newTestServer(newFakeUsers()).Handle(context.Background(), Request{Method: "OPTIONS", Path: "/v1/user"})

	if !strings.Contains(resp.Headers["Access-Control-Allow-Methods"], "PUT") {
		t.Errorf("Allow-Methods = %q, want it to include PUT", resp.Headers["Access-Control-Allow-Methods"])
	}
	if !strings.Contains(resp.Headers["Access-Control-Allow-Headers"], "Authorization") {
		t.Errorf("Allow-Headers = %q, want it to include Authorization", resp.Headers["Access-Control-Allow-Headers"])
	}
}

func TestPing(t *testing.T) {
	resp := newTestServer(newFakeUsers()).Ping()

	assertStatus(t, resp, 200)
	if body := decodeBody(t, resp); body["ok"] != true || body["timestamp"] == "" {
		t.Errorf("body = %v, want ok=true and a timestamp", body)
	}
}

// Both user endpoints share authorizeUserRequest, so exercise it through each method.
func TestUserEndpoints_Authorization(t *testing.T) {
	tests := []struct {
		name       string
		modify     func(*Request)
		noVerifier bool
		wantStatus int
	}{
		{name: "missing authorization header", modify: func(r *Request) { r.Headers = nil }, wantStatus: 401},
		{name: "malformed authorization header", modify: func(r *Request) { r.Headers["Authorization"] = testToken }, wantStatus: 401},
		{name: "invalid token", modify: func(r *Request) { r.Headers["Authorization"] = "Bearer bad" }, wantStatus: 401},
		{name: "auth not configured", noVerifier: true, wantStatus: 401},
		{name: "missing id", modify: func(r *Request) { r.QueryParams = nil }, wantStatus: 400},
		{name: "another user's id", modify: func(r *Request) { r.QueryParams["id"] = "someone-else" }, wantStatus: 403},
	}

	for _, method := range []string{"GET", "PUT"} {
		for _, tt := range tests {
			t.Run(method+" "+tt.name, func(t *testing.T) {
				users := newFakeUsers(&repositories.User{ID: testUserID}, &repositories.User{ID: "someone-else"})
				sc := ServiceContext{Repositories: repositories.Repositories{Users: users}}
				if !tt.noVerifier {
					sc.CognitoVerifier = fakeVerifier{}
				}

				req := authedRequest(method, testUserID, `{"display_name":"x"}`)
				if tt.modify != nil {
					tt.modify(&req)
				}

				resp := New(sc).Handle(context.Background(), req)
				assertStatus(t, resp, tt.wantStatus)
				if users.updatedID != "" {
					t.Errorf("rejected request still updated user %q", users.updatedID)
				}
			})
		}
	}
}

func TestUserEndpoints_AcceptLowercaseAuthorizationHeader(t *testing.T) {
	req := authedRequest("GET", testUserID, "")
	req.Headers = map[string]string{"authorization": "Bearer " + testToken}

	resp := newTestServer(newFakeUsers(&repositories.User{ID: testUserID})).Handle(context.Background(), req)
	assertStatus(t, resp, 200)
}

func TestGetUser(t *testing.T) {
	users := newFakeUsers(&repositories.User{ID: testUserID, DisplayName: "Quinn"})

	resp := newTestServer(users).Handle(context.Background(), authedRequest("GET", testUserID, ""))

	assertStatus(t, resp, 200)
	body := decodeBody(t, resp)
	want := map[string]any{"id": testUserID, "email": "user@example.com", "display_name": "Quinn"}
	if len(body) != len(want) {
		t.Errorf("body = %v, want exactly %v", body, want)
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("body[%q] = %v, want %v", k, body[k], v)
		}
	}
}

func TestGetUser_NotFound(t *testing.T) {
	resp := newTestServer(newFakeUsers()).Handle(context.Background(), authedRequest("GET", testUserID, ""))
	assertStatus(t, resp, 404)
}

func TestGetUser_RepositoryError(t *testing.T) {
	users := newFakeUsers()
	users.err = errors.New("boom")

	resp := newTestServer(users).Handle(context.Background(), authedRequest("GET", testUserID, ""))
	assertStatus(t, resp, 500)
}

func TestUpdateUser(t *testing.T) {
	users := newFakeUsers(&repositories.User{ID: testUserID})

	resp := newTestServer(users).Handle(context.Background(), authedRequest("PUT", testUserID, `{"display_name":"  New Name  "}`))

	assertStatus(t, resp, 200)
	if users.updatedID != testUserID || users.updatedName != "New Name" {
		t.Errorf("updated (%q, %q), want (%q, %q)", users.updatedID, users.updatedName, testUserID, "New Name")
	}
	body := decodeBody(t, resp)
	if body["display_name"] != "New Name" || body["email"] != "user@example.com" || body["id"] != testUserID {
		t.Errorf("body = %v", body)
	}
}

func TestUpdateUser_EmptyClearsDisplayName(t *testing.T) {
	users := newFakeUsers(&repositories.User{ID: testUserID, DisplayName: "Old"})

	resp := newTestServer(users).Handle(context.Background(), authedRequest("PUT", testUserID, `{"display_name":"   "}`))

	assertStatus(t, resp, 200)
	if users.updatedName != "" {
		t.Errorf("updated name = %q, want empty", users.updatedName)
	}
	if body := decodeBody(t, resp); body["display_name"] != "" {
		t.Errorf("display_name = %v, want empty", body["display_name"])
	}
}

func TestUpdateUser_BadRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "invalid JSON", body: "{"},
		{name: "missing display_name", body: `{}`},
		{name: "null display_name", body: `{"display_name":null}`},
		{name: "wrong type", body: `{"display_name":5}`},
		{name: "too long", body: `{"display_name":"` + strings.Repeat("a", maxDisplayNameLength+1) + `"}`},
		{name: "control character", body: `{"display_name":"a\nb"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users := newFakeUsers(&repositories.User{ID: testUserID})

			resp := newTestServer(users).Handle(context.Background(), authedRequest("PUT", testUserID, tt.body))

			assertStatus(t, resp, 400)
			if users.updatedID != "" {
				t.Error("invalid request still updated the user")
			}
		})
	}
}

func TestUpdateUser_NotFound(t *testing.T) {
	resp := newTestServer(newFakeUsers()).Handle(context.Background(), authedRequest("PUT", testUserID, `{"display_name":"x"}`))
	assertStatus(t, resp, 404)
}

func TestUpdateUser_RepositoryError(t *testing.T) {
	users := newFakeUsers(&repositories.User{ID: testUserID})
	users.err = errors.New("boom")

	resp := newTestServer(users).Handle(context.Background(), authedRequest("PUT", testUserID, `{"display_name":"x"}`))
	assertStatus(t, resp, 500)
}

func TestValidateDisplayName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "plain", input: "Quinn", want: "Quinn"},
		{name: "trims whitespace", input: "  Quinn \t", want: "Quinn"},
		{name: "empty clears", input: "", want: ""},
		{name: "whitespace only clears", input: "   ", want: ""},
		{name: "max length", input: strings.Repeat("a", maxDisplayNameLength), want: strings.Repeat("a", maxDisplayNameLength)},
		{name: "over max length", input: strings.Repeat("a", maxDisplayNameLength+1), wantErr: true},
		// Length is counted in characters, not bytes.
		{name: "multibyte at max length", input: strings.Repeat("é", maxDisplayNameLength), want: strings.Repeat("é", maxDisplayNameLength)},
		{name: "length checked after trimming", input: " " + strings.Repeat("a", maxDisplayNameLength) + " ", want: strings.Repeat("a", maxDisplayNameLength)},
		{name: "unicode and emoji", input: "Zoë 🚀", want: "Zoë 🚀"},
		{name: "inner newline", input: "a\nb", wantErr: true},
		{name: "null byte", input: "a\x00b", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateDisplayName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	resp := errorResponse(418, errors.New(`bad "quote"`))

	assertStatus(t, resp, 418)
	if body := decodeBody(t, resp); body["error"] != `bad "quote"` {
		t.Errorf("error = %v, want the message JSON-escaped and round-tripped", body["error"])
	}
}
