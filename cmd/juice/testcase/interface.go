package testcase

import (
	"context"
	"os/user"
)

//go:generate juice --type Interface --config config.xml --namespace main.UserRepository --output interface_impl.go
type Interface interface {
	GetUserByID(ctx context.Context, id int64) (*User, error)
	CreateUser(ctx context.Context, u *user.User) error
}

type User struct{}
