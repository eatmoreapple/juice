## Juice SQL Mapper Framework For Golang

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

<configuration>
    <environments default="prod">
        <environment id="prod">
            <dataSource>root:qwe123@tcp(localhost:3306)/database</dataSource>
            <driver>mysql</driver>
        </environment>
    </environments>


    <mappers>
        <mapper namespace="main.UserRepository">
            <select id="Count">
                select count(*) from user
            </select>

            <select id="GetUserByID">
                select * from user where id = #{id}
            </select>

            <update id="UpdateUser">
                update user set name = #{name}
                <if test="age > 0">
                    ,age = #{age}
                </if>
                where id = #{id}
            </update>

            <delete id="DeleteUserByID">
                delete from user where id = #{id}
            </delete>

            <insert id="CreateUser">
                insert into user (`name`, `age`) values (#{name}, #{age})
            </insert>

            <insert id="BatchCreateUser">
                insert into user (`name`, `age`) values
                <foreach collection="params" item="user" separator=", ">
                    (#{user.name}, #{user.age})
                </foreach>
            </insert>

        </mapper>
    </mappers>
</configuration>

```

```go
package main

import (
	"fmt"

	"github.com/eatmoreapple/juice"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	Id   int64  `column:"id" param:"id"`
	Name string `column:"name" param:"name"`
	Age  int    `column:"age" param:"age"`
}

type UserRepository interface {
	Count() (int64, error)
	GetUserByID(user *User) (*User, error)
	UpdateUser(user *User) (int64, error)
	DeleteUserByID(user *User) (int64, error)
	CreateUser(user *User) (int64, error)
	BatchCreateUser(users []*User) (int64, error)
}

type UserRepositoryImpl struct{}

func (u UserRepositoryImpl) Count() (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) GetUserByID(user *User) (*User, error) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) UpdateUser(user *User) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) DeleteUserByID(user *User) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) CreateUser(user *User) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) BatchCreateUser(users []*User) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func main() {

	cfg, err := juice.NewXMLConfiguration("config.xml")
	if err != nil {
		panic(err)
	}
	engine, err := juice.DefaultEngine(cfg)
	if err != nil {
		panic(err)
	}

	var repo UserRepository = UserRepositoryImpl{}

	fmt.Println(juice.NewGenericEngine[int, any](engine).Statement(repo.Count).Query(nil).One())

	var user = User{Id: 1, Name: "eatmoreapple", Age: 18}

	fmt.Println(juice.NewGenericEngine[int, *User](engine).Statement(repo.GetUserByID).Query(&user).One())
    
	// Using Transaction
	tx := engine.Tx()

	result, err := tx.Statement(repo.CreateUser).Exec(&user)
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
	}

	user.Id, err = result.LastInsertId()

	if err != nil {
		fmt.Println(err)
		tx.Rollback()
	}

	if _, err = tx.Statement(repo.DeleteUserByID).Exec(&user); err != nil {
		fmt.Println(err)
		tx.Rollback()
	}

	if err = tx.Commit(); err != nil {
		fmt.Println(err)
		tx.Rollback()
	}

	fmt.Println(juice.NewGenericEngine[int, *User](engine).Statement(repo.UpdateUser).Exec(&user))

	fmt.Println(juice.NewGenericEngine[int, []*User](engine).Statement(repo.BatchCreateUser).Exec([]*User{&user}))

}

```

### License

Juice is licensed under the Apache License, Version 2.0. See LICENSE for the full license text.

### Contact

If you have any questions, please contact me by wechat: eatmoreapple

if you like this project, please give me a star, thank you very much.