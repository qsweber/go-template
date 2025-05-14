build::
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bootstrap ./handler/handler.go
	zip -j ./handler.zip ./bootstrap
