package testcase

import (
	"context"
	"database/sql"
	"os/user"
)

//go:generate juice --type Interface --config config.xml --namespace main.UserRepository --output interface_impl.go
type Interface interface {
	GetUserByID(ctx context.Context, id int64) (*User, error)
	CreateUser(ctx context.Context, u *user.User) error
	DeleteUserByID(ctx context.Context, id int64) (sql.Result, error)
}

type User struct{}
