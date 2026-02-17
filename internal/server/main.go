package server

type Server interface {
	Ping() bool
	Foo(input FooInput) (FooOutput, error)
}

type ServerImpl struct{}

func New() Server {
	return &ServerImpl{}
}
