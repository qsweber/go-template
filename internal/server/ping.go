package server

type PingOutput struct {
	Ok bool `json:"ok"`
}

func (s *ServerImpl) Ping() (PingOutput, error) {
	return PingOutput{Ok: true}, nil
}
