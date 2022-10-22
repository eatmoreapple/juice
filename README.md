## Juice SQL Mapper Framework For Golang

![Go Doc](https://pkg.go.dev/badge/github.com/eatmoreapple/juice)
![Go Report Card](https://goreportcard.com/badge/github.com/eatmoreapple/juice)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

This is a SQL mapper framework for Golang. It is inspired by MyBatis.

Juice is a simple and lightweight framework. It is easy to use and easy to extend.

### Features

* Simple and lightweight
* Easy to use
* Easy to extend
* Support for multiple databases
* Dynamic SQL
* Result to entity mapping
* Generic type support
* Middleware support
* Todo support more

### Quick Start

#### Install

```bash
go get github.com/eatmoreapple/juice
```

#### Example

```shell
touch config.xml
```

and write the following content into config.xml

```xml
<?xml version="1.0" encoding="UTF-8"?>
<configuration>
    <environments default="prod">
        <environment id="prod">
            <dataSource>root:qwe123@tcp(localhost:3306)/database</dataSource>
            <driver>mysql</driver>
        </environment>
    </environments>


    <mappers>
        <mapper namespace="main.UserRepository">

            <select id="GetUserByID" debug="true">
                select * from user where id = #{param}
            </select>

            <insert id="CreateUser" debug="true">
                insert into user (`name`, `age`) values (#{name}, #{age})
            </insert>

            <update id="UpdateUser" debug="true">
                update user set `name` = #{name}, `age` = #{age} where id = #{id}
            </update>

            <delete id="DeleteUserByID" debug="true">
                delete from user where id = #{param}
            </delete>

        </mapper>
    </mappers>
</configuration>
```

```go
package main

import (
	"context"
	"fmt"
	"github.com/eatmoreapple/juice"
	"reflect"

	_ "github.com/go-sql-driver/mysql"
)

var schema = `
CREATE TABLE IF NOT EXISTS user (
  id int(11) NOT NULL AUTO_INCREMENT,
  name varchar(255) COLLATE utf8mb4_bin NOT NULL,
  age int(11) NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
`

type User struct {
	Id   int64  `column:"id" param:"id"`
	Name string `column:"name" param:"name"`
	Age  int    `column:"age" param:"age"`
}

type UserRepository interface {
	GetUserByID(id int64)
	CreateUser(user *User)
	UpdateUser(user *User)
	DeleteUserByID(id int64)
}

type UserRepositoryImpl struct{}

func (u UserRepositoryImpl) UpdateUser(user *User) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) DeleteUserByID(id int64) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) CreateUser(user *User) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) GetUserByID(id int64) {
	//TODO implement me
	panic("implement me")
}

func main() {
	cfg, err := juice.NewXMLConfiguration("config.xml")
	if err != nil {
		fmt.Println(err)
		return
	}

	engine, err := juice.DefaultEngine(cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	// create table first
	if _, err = engine.DB.Exec(schema); err != nil {
		fmt.Println(err)
		return
	}

	var repo UserRepository = UserRepositoryImpl{}

	// create user first
	user := &User{
		Name: "eatmoreapple",
		Age:  18,
	}

	result, err := engine.Object(repo.CreateUser).ExecContext(context.Background(), user)
	if err != nil {
		fmt.Println(err)
		return
	}
	user.Id, err = result.LastInsertId()
	if err != nil {
		fmt.Println(err)
		return
	}

	// query user
	rows, err := engine.Object(repo.GetUserByID).Query(user.Id)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()

	// todo: iterate rows with your own way

	// query user with auto mapping
	user2, err := juice.NewGenericManager[*User](engine).Object(repo.GetUserByID).Query(user.Id).One()
	if err != nil {
		fmt.Println(err)
		return
	}

	if reflect.DeepEqual(user, user2) {
		fmt.Println("user and user2 are equal")
	} else {
		fmt.Println("user and user2 are not equal")
		return
	}

	// with transaction

	// begin transaction

	// begin
	tx := engine.Tx()

	// update user with tx
	user.Name = "eatmoreapple2"
	user.Age = 20
	result, err = tx.Object(repo.UpdateUser).ExecContext(context.Background(), user)
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
		return
	}
	fmt.Printf("update affected rows: %d\n", affected)

	// delete user with tx
	result, err = tx.Object(repo.DeleteUserByID).ExecContext(context.Background(), user.Id)
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
		return
	}
	affected, err = result.RowsAffected()
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
		return
	}
	fmt.Printf("delete affected rows: %d\n", affected)

	// commit
	if err = tx.Commit(); err != nil {
		fmt.Println(err)
		return
	}
}
```

### License

Juice is licensed under the Apache License, Version 2.0. See LICENSE for the full license text.

### Contact

If you like this project, please give me a star. Thank you.
And If you have any questions, please contact me by WeChat: eatmoreapple or scan the following QR code.


<img width="210px"  src="https://github.com/eatmoreapple/eatMoreApple/blob/main/img/wechat.jpg" align="left">
