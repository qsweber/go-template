package auth

import (
	"fmt"
	"os"
)

// GetCognitoConfig loads Cognito configuration from environment variables
func GetCognitoConfig() (CognitoConfig, error) {
	region := os.Getenv("COGNITO_REGION")
	if region == "" {
		return CognitoConfig{}, fmt.Errorf("COGNITO_REGION environment variable is required")
	}

	userPoolID := os.Getenv("COGNITO_USER_POOL_ID")
	if userPoolID == "" {
		return CognitoConfig{}, fmt.Errorf("COGNITO_USER_POOL_ID environment variable is required")
	}

	clientID := os.Getenv("COGNITO_CLIENT_ID")
	if clientID == "" {
		return CognitoConfig{}, fmt.Errorf("COGNITO_CLIENT_ID environment variable is required")
	}

	return CognitoConfig{
		Region:     region,
		UserPoolID: userPoolID,
		ClientID:   clientID,
	}, nil
}
