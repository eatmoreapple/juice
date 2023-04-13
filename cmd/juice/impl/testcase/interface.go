package testcase2

import (
	"context"
	"database/sql"
	"fmt"
	"os/user"
	"reflect"
	"runtime"
)

//go:generate juice impl --type Interface  --output interface_impl.go
type Interface interface {
	// GetUserByID 根据用户id查找用户
	GetUserByID(ctx context.Context, id int64) ([]*user.User, error)
	// CreateUser 创建用户
	CreateUser(ctx context.Context, u map[string]*user.User) error
	// DeleteUserByID 根据id删除用户
	DeleteUserByID(ctx context.Context, id int64) (sql.Result, error)
}

type User struct{}

func main() {
	var a Interface = NewInterface()
	fmt.Println(runtime.FuncForPC(reflect.ValueOf(a.GetUserByID).Pointer()).Name())
}
