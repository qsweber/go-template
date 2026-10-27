package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testRegion     = "us-west-2"
	testUserPoolID = "us-west-2_TestPool"
	testClientID   = "test-client-id"
	testKid        = "test-kid"
)

var testIssuer = "https://cognito-idp." + testRegion + ".amazonaws.com/" + testUserPoolID

type testJWKSServer struct {
	key      *rsa.PrivateKey
	server   *httptest.Server
	fetches  atomic.Int32
	status   int
	verifier *CognitoVerifier
}

// newTestJWKSServer serves a JWKS containing one RSA key and returns a verifier pointed at it.
func newTestJWKSServer(t *testing.T) *testJWKSServer {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	ts := &testJWKSServer{key: key, status: http.StatusOK}
	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.fetches.Add(1)
		if ts.status != http.StatusOK {
			w.WriteHeader(ts.status)
			return
		}
		_ = json.NewEncoder(w).Encode(JWKS{Keys: []JWK{{
			Alg: "RS256",
			Kty: "RSA",
			Use: "sig",
			Kid: testKid,
			N:   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
		}}})
	}))
	t.Cleanup(ts.server.Close)

	ts.verifier = NewCognitoVerifier(CognitoConfig{Region: testRegion, UserPoolID: testUserPoolID, ClientID: testClientID})
	ts.verifier.jwksURL = ts.server.URL

	return ts
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"sub":       "user-123",
		"iss":       testIssuer,
		"aud":       testClientID,
		"token_use": "id",
		"email":     "user@example.com",
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func TestVerifyToken_Valid(t *testing.T) {
	ts := newTestJWKSServer(t)

	claims, err := ts.verifier.VerifyToken(context.Background(), signToken(t, ts.key, testKid, validClaims()))
	if err != nil {
		t.Fatalf("expected valid token, got error: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "user@example.com")
	}
}

func TestVerifyToken_Rejects(t *testing.T) {
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	tests := []struct {
		name      string
		mutate    func(jwt.MapClaims)
		kid       string
		signWith  *rsa.PrivateKey // nil means the JWKS key
		wantError string
	}{
		{name: "wrong issuer", mutate: func(c jwt.MapClaims) { c["iss"] = "https://example.com" }, wantError: "invalid issuer"},
		{name: "wrong audience", mutate: func(c jwt.MapClaims) { c["aud"] = "other-client" }, wantError: "invalid audience"},
		// Cognito access tokens have no aud claim; this used to panic on Audience[0].
		{name: "missing audience", mutate: func(c jwt.MapClaims) { delete(c, "aud") }, wantError: "invalid audience"},
		{name: "access token", mutate: func(c jwt.MapClaims) { c["token_use"] = "access" }, wantError: "invalid token_use"},
		{name: "expired", mutate: func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() }, wantError: "expired"},
		// Without exp the parser skips its own expiry check; this used to panic on ExpiresAt.
		{name: "missing expiration", mutate: func(c jwt.MapClaims) { delete(c, "exp") }, wantError: "expired"},
		{name: "unknown kid", kid: "other-kid", wantError: "public key not found"},
		{name: "signed by another key", signWith: otherKey, wantError: "failed to parse token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTestJWKSServer(t)

			claims := validClaims()
			if tt.mutate != nil {
				tt.mutate(claims)
			}
			kid := testKid
			if tt.kid != "" {
				kid = tt.kid
			}
			key := ts.key
			if tt.signWith != nil {
				key = tt.signWith
			}

			_, err := ts.verifier.VerifyToken(context.Background(), signToken(t, key, kid, claims))
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestVerifyToken_RejectsNonRSASigningMethod(t *testing.T) {
	ts := newTestJWKSServer(t)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims())
	token.Header["kid"] = testKid
	signed, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = ts.verifier.VerifyToken(context.Background(), signed)
	if err == nil || !strings.Contains(err.Error(), "unexpected signing method") {
		t.Errorf("error = %v, want unexpected signing method", err)
	}
}

func TestVerifyToken_RejectsMalformedToken(t *testing.T) {
	ts := newTestJWKSServer(t)

	if _, err := ts.verifier.VerifyToken(context.Background(), "not-a-jwt"); err == nil {
		t.Error("expected an error for a malformed token, got nil")
	}
}

func TestVerifyToken_JWKSFetchFailure(t *testing.T) {
	ts := newTestJWKSServer(t)
	ts.status = http.StatusInternalServerError

	_, err := ts.verifier.VerifyToken(context.Background(), signToken(t, ts.key, testKid, validClaims()))
	if err == nil || !strings.Contains(err.Error(), "failed to fetch JWKS") {
		t.Errorf("error = %v, want failed to fetch JWKS", err)
	}
}

func TestVerifyToken_CachesJWKS(t *testing.T) {
	ts := newTestJWKSServer(t)
	token := signToken(t, ts.key, testKid, validClaims())

	for i := 0; i < 3; i++ {
		if _, err := ts.verifier.VerifyToken(context.Background(), token); err != nil {
			t.Fatalf("verify %d: unexpected error: %v", i, err)
		}
	}
	if got := ts.fetches.Load(); got != 1 {
		t.Errorf("JWKS fetched %d times, want 1", got)
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{name: "valid", header: "Bearer abc.def.ghi", want: "abc.def.ghi"},
		{name: "case-insensitive scheme", header: "bearer abc", want: "abc"},
		{name: "empty", header: "", wantErr: true},
		{name: "missing token", header: "Bearer", wantErr: true},
		{name: "wrong scheme", header: "Basic abc", wantErr: true},
		{name: "too many parts", header: "Bearer abc def", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractBearerToken(tt.header)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("token = %q, want %q", got, tt.want)
			}
		})
	}
}
