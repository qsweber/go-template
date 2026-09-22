package server

import (
	"encoding/json"
	"time"
)

type pingOutput struct {
	Ok        bool   `json:"ok"`
	Timestamp string `json:"timestamp"`
}

func (s *ServerImpl) Ping() Response {
	body, err := json.Marshal(pingOutput{
		Ok:        true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return errorResponse(500, err)
	}

	return Response{
		StatusCode: 200,
		Body:       string(body),
		Headers:    corsHeaders(),
	}
}
