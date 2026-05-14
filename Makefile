build-apigateway-lambda::
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bootstrap ./cmd/apigateway/main.go
	zip -j ./handler.zip ./bootstrap

build-dynamostream-lambda::
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bootstrap-stream ./cmd/dynamostream/main.go
	zip -j ./stream.zip ./bootstrap-stream
