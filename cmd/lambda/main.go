package main

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/qsweber/go-template/internal/server"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	srv := server.New()

	switch request.Path {
	case "/ping":
		result := srv.Ping()
		responseBytes, err := json.Marshal(map[string]bool{"ok": result})
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error encoding response"}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       string(responseBytes),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	case "/foo":
		bar := request.QueryStringParameters["bar"] // /foo?bar=hello
		input := server.FooInput{Bar: bar}
		output, err := srv.Foo(input)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error calling Foo"}, err
		}
		responseBytes, err := json.Marshal(output)
		if err != nil {
			return events.APIGatewayProxyResponse{StatusCode: 500, Body: "Error encoding response"}, err
		}
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       string(responseBytes),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	default:
		return events.APIGatewayProxyResponse{StatusCode: 404, Body: "Not Found"}, nil
	}
}

func main() {
	lambda.Start(handler)
}
