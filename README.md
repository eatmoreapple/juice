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
            <select id="GetUserByID">
                select * from user where id = #{params}
            </select>
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
	GetUserByID(id int64) (*User, error)
}

type UserRepositoryImpl struct{}

func (u UserRepositoryImpl) GetUserByID(id int64) (*User, error) {
	//TODO implement me
	panic("implement me")
}

func main() {
	cfg, err := juice.NewXMLConfiguration("/Users/eatmoreapple/GolandProjects/pillow/.example/config.xml")
	if err != nil {
		panic(err)
	}

	engine, err := juice.DefaultEngine(cfg)
	if err != nil {
		panic(err)
	}

	var repo UserRepository = UserRepositoryImpl{}

	user, err := juice.NewGenericManager[*User](engine).Object(repo.GetUserByID).Query(3).One()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v", user)
}


```

### License

Juice is licensed under the Apache License, Version 2.0. See LICENSE for the full license text.

### Contact

If you like this project, please give me a star. Thank you.
And If you have any questions, please contact me by WeChat: eatmoreapple or scan the following QR code.


<img width="210px"  src="https://github.com/eatmoreapple/eatMoreApple/blob/main/img/wechat.jpg" align="left">
