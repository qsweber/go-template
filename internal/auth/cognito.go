package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CognitoConfig holds the configuration for Cognito authentication
type CognitoConfig struct {
	Region     string
	UserPoolID string
	ClientID   string
}

// JWK represents a JSON Web Key
type JWK struct {
	Alg string `json:"alg"`
	E   string `json:"e"`
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	Use string `json:"use"`
}

// JWKS represents a set of JSON Web Keys
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// CognitoVerifier verifies Cognito JWT tokens
type CognitoVerifier struct {
	config     CognitoConfig
	jwksURL    string
	keys       map[string]*rsa.PublicKey
	lastFetch  time.Time
	httpClient *http.Client
}

// Claims represents the JWT claims
type Claims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	CognitoUser   string `json:"cognito:username"`
	TokenUse      string `json:"token_use"`
}

// NewCognitoVerifier creates a new Cognito JWT verifier
func NewCognitoVerifier(config CognitoConfig) *CognitoVerifier {
	jwksURL := fmt.Sprintf(
		"https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json",
		config.Region,
		config.UserPoolID,
	)

	return &CognitoVerifier{
		config:     config,
		jwksURL:    jwksURL,
		keys:       make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// fetchJWKS fetches the public keys from Cognito
func (v *CognitoVerifier) fetchJWKS(ctx context.Context) error {
	// Only fetch if we haven't fetched recently (cache for 1 hour)
	if time.Since(v.lastFetch) < time.Hour && len(v.keys) > 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert JWKs to RSA public keys
	newKeys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		pubKey, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			return fmt.Errorf("failed to convert JWK to RSA key: %w", err)
		}
		newKeys[jwk.Kid] = pubKey
	}

	v.keys = newKeys
	v.lastFetch = time.Now()

	return nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key
func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode the modulus
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode the exponent
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	pubKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	return pubKey, nil
}

// VerifyToken verifies a Cognito JWT token and returns the claims
func (v *CognitoVerifier) VerifyToken(ctx context.Context, tokenString string) (*Claims, error) {
	// Fetch JWKS if needed
	if err := v.fetchJWKS(ctx); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid header not found")
		}

		// Get the public key
		pubKey, ok := v.keys[kid]
		if !ok {
			return nil, fmt.Errorf("public key not found for kid: %s", kid)
		}

		return pubKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Verify the issuer
	expectedIssuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", v.config.Region, v.config.UserPoolID)
	if claims.Issuer != expectedIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", expectedIssuer, claims.Issuer)
	}

	// Verify the audience (client ID)
	if claims.Audience[0] != v.config.ClientID {
		return nil, fmt.Errorf("invalid audience: expected %s, got %s", v.config.ClientID, claims.Audience)
	}

	// Verify token use (should be "id" for ID tokens)
	if claims.TokenUse != "id" {
		return nil, fmt.Errorf("invalid token_use: expected id, got %s", claims.TokenUse)
	}

	// Verify expiration
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, errors.New("token has expired")
	}

	return claims, nil
}

// ExtractBearerToken extracts the bearer token from the Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("authorization header must be in format: Bearer <token>")
	}

	return parts[1], nil
}
