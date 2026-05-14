package server

import "context"

type FooInput struct {
	Bar string
}

type FooOutput struct {
	Baz string `json:"baz"`
}

func (s *ServerImpl) Foo(ctx context.Context, cognitoUserID string, input FooInput) (FooOutput, error) {
	// Record the click
	if err := s.clicksRepository.RecordClick(ctx, cognitoUserID); err != nil {
		return FooOutput{}, err
	}

	return FooOutput{Baz: input.Bar}, nil
}
