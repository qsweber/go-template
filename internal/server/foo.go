package server

type FooInput struct {
	Bar string
}

type FooOutput struct {
	Baz string `json:"baz"`
}

func (s *ServerImpl) Foo(input FooInput) (FooOutput, error) {
	return FooOutput{Baz: input.Bar}, nil
}
