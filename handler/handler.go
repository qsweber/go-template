package main

import (
	"encoding/json"
	"log"

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

	// Prepare response as JSON
	response := map[string]string{
		"result": request.Path[1:],
	}
	responseBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Internal Server Error"}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(responseBytes),
	}, nil
}

func main() {
	lambda.Start(handler)
}
