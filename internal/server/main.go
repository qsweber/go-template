package server

import (
	"context"

	"github.com/qsweber/go-template/internal/clicks"
)

type Server interface {
	Ping() (PingOutput, error)
	Foo(ctx context.Context, cognitoUserID string, input FooInput) (FooOutput, error)
}

type ServerImpl struct {
	clicksRepository clicks.Repository
}

func New(clicksRepository clicks.Repository) Server {
	return &ServerImpl{
		clicksRepository: clicksRepository,
	}
}
