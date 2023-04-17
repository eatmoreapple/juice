package main

import (
	"context"
	"database/sql"
)

//go:generate juice impl --type Interface  --output interface_impl.go
type Interface interface {
	// GetUserByID 根据用户id查找用户
	GetUserByID(ctx context.Context, id int64) ([]*User, error)
	// CreateUser 创建用户
	CreateUser(ctx context.Context, u map[string]*User) error
	// DeleteUserByID 根据id删除用户
	DeleteUserByID(ctx context.Context, id int64) (sql.Result, error)
}

type User struct{}
