build::
	GOOS=linux GOARCH=amd64 go build -o ./build/bootstrap ./handler/handler.go
	zip -j ./build/handler.zip ./build/bootstrap
