package auth

import "testing"

func TestGetCognitoConfig(t *testing.T) {
	t.Setenv("COGNITO_REGION", "us-west-2")
	t.Setenv("COGNITO_USER_POOL_ID", "pool")
	t.Setenv("COGNITO_CLIENT_ID", "client")

	config, err := GetCognitoConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := CognitoConfig{Region: "us-west-2", UserPoolID: "pool", ClientID: "client"}
	if config != want {
		t.Errorf("config = %+v, want %+v", config, want)
	}
}

func TestGetCognitoConfig_MissingVariable(t *testing.T) {
	for _, missing := range []string{"COGNITO_REGION", "COGNITO_USER_POOL_ID", "COGNITO_CLIENT_ID"} {
		t.Run(missing, func(t *testing.T) {
			t.Setenv("COGNITO_REGION", "us-west-2")
			t.Setenv("COGNITO_USER_POOL_ID", "pool")
			t.Setenv("COGNITO_CLIENT_ID", "client")
			t.Setenv(missing, "")

			if _, err := GetCognitoConfig(); err == nil {
				t.Errorf("expected an error when %s is unset", missing)
			}
		})
	}
}
