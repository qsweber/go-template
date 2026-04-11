package server

type Server interface {
	Ping() (PingOutput, error)
	Foo(input FooInput) (FooOutput, error)
}

type ServerImpl struct{}

func New() Server {
	return &ServerImpl{}
}
