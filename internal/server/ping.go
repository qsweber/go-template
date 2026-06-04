package server

func (s *ServerImpl) Ping() Response {
	return Response{
		StatusCode: 200,
		Body:       `{"ok": true}`,
		Headers:    corsHeaders(),
	}
}
