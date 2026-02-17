package server

type FooInput struct {
	Bar string
}

type FooOutput struct {
	Baz string
}

func (s *ServerImpl) Foo(input FooInput) (FooOutput, error) {
	return FooOutput{Baz: input.Bar + " world"}, nil
}
