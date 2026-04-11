# Go Template

This repository contains:

- A local HTTP API entrypoint at cmd/api/main.go
- An AWS Lambda entrypoint at cmd/lambda/main.go
- Shared business logic in internal/server
- Shared request routing in internal/rpc
- Pulumi infrastructure code in infrastructure

## Prerequisites

1. Go 1.24+
2. (Optional, for deployment) Pulumi CLI and AWS credentials

## Run The API Locally

Start the local API server from the repository root:

```bash
go run ./cmd/api/main.go
```

The server listens on port 8080.

### Endpoints

1. Health check:

```bash
curl -i http://localhost:8080/ping
```

2. Protected foo route without auth (expected 401):

```bash
curl -i http://localhost:8080/foo
```

3. Protected foo route with mock auth (expected 200):

```bash
curl -i -H "Authorization: Bearer dev-token" http://localhost:8080/foo
```

Expected body for successful foo request:

```json
{"baz": "example"}
```

## Local Auth Behavior

For local development, cmd/api/main.go uses a mock token verifier.

- Any request to /foo must still include an Authorization header in Bearer format.
- The token value is not validated against Cognito in local mode.

This keeps local development simple while preserving authenticated request flow.

## Lambda Auth Behavior

The Lambda entrypoint uses Cognito verification when these environment variables are set:

- COGNITO_REGION
- COGNITO_USER_POOL_ID
- COGNITO_CLIENT_ID

If configuration is missing, auth verification is disabled in Lambda initialization.

## Build Lambda Artifact

Use the included make target:

```bash
make build-lambda
```

This produces:

- bootstrap
- handler.zip

## Deploy With Pulumi

1. Initialize/select a stack:

```bash
pulumi stack init dev
```

2. Set AWS region:

```bash
pulumi config set aws:region us-west-2
```

3. Deploy:

```bash
pulumi up
```

4. Tear down when done:

```bash
pulumi destroy --yes
pulumi stack rm --yes
```
