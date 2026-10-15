build-apigateway-lambda::
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bootstrap ./cmd/apigateway/main.go
	zip -j ./handler.zip ./bootstrap

build-dynamostream-lambda::
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bootstrap ./cmd/dynamostream/main.go
	rm -f ./stream.zip
	zip -j ./stream.zip ./bootstrap

build-postconfirmation-lambda::
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bootstrap ./cmd/postconfirmation/main.go
	rm -f ./postconfirmation.zip
	zip -j ./postconfirmation.zip ./bootstrap
