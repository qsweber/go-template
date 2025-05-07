package main

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// handler is a simple function that takes a string and does a ToUpper.
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// log the request
	log.Println("Request: ", request)
	// marshal the request to a string
	requestBytes, _ := json.Marshal(request)
	requestStr := string(requestBytes)
	log.Println("Request String: ", requestStr)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       strings.ToUpper(request.Path[1:]),
	}, nil
}

func main() {
	lambda.Start(handler)
}
