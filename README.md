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

### support xml tags

* select
* insert
* update
* delete
* sql
* if
* where
* trim
* set
* foreach
* choose
* when
* otherwise
* include


### Condition Method
The condition method can be used with `if` or `when` tags.

For example:
```xml
<!--ids = []int{1,2,3}-->
<if test='len(ids) > 0 && substr("eatmoreapple", 0, 3) == "eat"'>
    your sql node here
</if>
```

It can register to the framework with your own condition method.

Here are some default condition methods.

* len: return the length of the given parameter
* strsub: return the substring of the given parameter
* join: join the given parameters with the given separator
* contains: return true if the given parameter contains the given element
* slice: return the slice of the given parameter

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
        <mapper namespace="main.UserRepository" table="user">

            <sql id="columns">
                `id`, `name`, `age`
            </sql>

            <sql id="selectAll">
                select
                <include refid="columns"/>
                from ${table}
            </sql>


            <select id="GetUserByID" debug="true">
                <include refid="selectAll"/>
                where id = #{param}
            </select>

            <insert id="CreateUser" debug="true">
                insert into ${table}
                <trim prefix="(" suffixOverrides="," suffix=")">
                    <if test='name != ""'>
                        name,
                    </if>
                    <if test="age > 0">
                        age,
                    </if>
                </trim>
                <trim prefix="values (" suffix=")" suffixOverrides=",">
                    <if test='name != ""'>
                        #{name},
                    </if>
                    <if test="age > 0">
                        #{age},
                    </if>
                </trim>
            </insert>

            <update id="UpdateUser" debug="true">
                update ${table}
                <set>
                    <if test='name != ""'>
                        `name` = #{name},
                    </if>
                    <if test="age > 0">
                        `age` = #{age},
                    </if>
                </set>
                where id = #{id}
            </update>

            <delete id="DeleteUserByID" debug="true" table="user">
                delete from ${table} where id = #{param}
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
	Create(user *User)
	UpdateUser(user *User)
	DeleteUserByID(id int64)
	List()
}

type UserRepositoryImpl struct{}

func (u UserRepositoryImpl) List() {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) UpdateUser(user *User) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) DeleteUserByID(id int64) {
	//TODO implement me
	panic("implement me")
}

func (u UserRepositoryImpl) Create(user *User) {
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
	if _, err = engine.DB().Exec(schema); err != nil {
		fmt.Println(err)
		return
	}

	var repo UserRepository = UserRepositoryImpl{}

	// create user first
	user := &User{
		Name: "eatmoreapple",
		Age:  18,
	}

	result, err := engine.Object(repo.Create).ExecContext(context.Background(), user)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(result.RowsAffected())

	users, err := juice.NewGenericManager[[]*User](engine).Object(repo.List).Query(nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, user := range users {
		fmt.Printf("%+v\n", user)
	}
}
```

### License

Juice is licensed under the Apache License, Version 2.0. See LICENSE for the full license text.

### Contact

If you like this project, please give me a star. Thank you.
And If you have any questions, please contact me by WeChat: eatmoreapple or scan the following QR code.


<img width="210px"  src="https://github.com/eatmoreapple/eatMoreApple/blob/main/img/wechat.jpg" align="left">
