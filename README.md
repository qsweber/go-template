# Go Template

This repository contains:

- A local HTTP API entrypoint at cmd/api/main.go
- An AWS Lambda entrypoint at cmd/apigateway/main.go
- An AWS Lambda stream consumer at cmd/dynamostream/main.go
- Shared business logic in internal/server
- Shared request routing in internal/rpc
- AWS CDK (Go) infrastructure code in cdk

## Prerequisites

1. Go 1.25.8+
2. (Optional, for deployment) Node.js 20+, the AWS CDK CLI, and AWS credentials

## Run The API Locally

Start the local API server from the repository root:

```bash
go run ./cmd/api/main.go
```

The server listens on port 8080.

### Endpoints

1. Health check:

```bash
curl -i http://localhost:8080/v1/ping
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
make build-apigateway-lambda
```

This produces:

- bootstrap
- handler.zip

For the DynamoDB stream consumer, use:

```bash
make build-dynamostream-lambda
```

This produces:

- bootstrap
- stream.zip

## Deploy With CDK

The `cdk/` directory is a standalone Go module containing a CDK app that defines two stacks,
`go-template-dev` and `go-template-prod`, each targeting `us-west-2`.

1. Install the pinned CDK CLI (only needed once):

```bash
cd cdk && npm install
```

2. One-time per AWS account/region, bootstrap the CDK toolkit:

```bash
npx cdk bootstrap aws://<account-id>/us-west-2
```

3. Build the Lambda artifacts (from the repo root) so the CDK app has something to package:

```bash
make build-apigateway-lambda
make build-dynamostream-lambda
```

4. Review and deploy a stack:

```bash
cd cdk
npx cdk diff go-template-dev
npx cdk deploy go-template-dev
```

Swap `go-template-dev` for `go-template-prod` to deploy the other environment.

5. Tear down when done:

```bash
npx cdk destroy go-template-dev
```

Note: the Cognito user pool/client, SES domain identity, and its Route53 DNS records are
deployed with a `RETAIN` removal policy, so `cdk destroy` will leave them in place.
