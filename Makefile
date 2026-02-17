build-lambda::
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bootstrap ./cmd/lambda/main.go
	zip -j ./handler.zip ./bootstrap
