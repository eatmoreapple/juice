## Juice SQL Mapper Framework For Golang

[![Go Doc](https://pkg.go.dev/badge/github.com/eatmoreapple/juice)](https://godoc.org/github.com/eatmoreapple/juice)
[![Release](https://img.shields.io/github/v/release/eatmoreapple/juice.svg?style=flat-square)](https://github.com/eatmoreapple/juice/releases)
![Go Report Card](https://goreportcard.com/badge/github.com/eatmoreapple/juice)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

Juice is a SQL mapper framework for Golang, inspired by MyBatis. It is simple, lightweight, and easy to use and extend.
This document provides a brief introduction to Juice and its usage.

- [Installation](#installation)
- [Example](#example)
- [API Documentation](#api-documentation)
- [License](#license)
- [Support Me](#support-me)

### Installation

To install Juice, use the following command:

```shell
go get github.com/eatmoreapple/juice
```

### Example

```shell
touch config.xml
```

add the following content to config.xml

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE configuration PUBLIC "-//juice.org//DTD Config 1.0//EN"
        "https://raw.githubusercontent.com/eatmoreapple/juice/main/config.dtd">

<configuration>
    <environments default="prod">
        <environment id="prod">
            <dataSource>root:qwe123@tcp(localhost:3306)/database</dataSource>
            <driver>mysql</driver>
        </environment>
    </environments>

    <mappers>
        <mapper resource="mappers.xml"/>
    </mappers>
</configuration>
```

```shell
touch mappers.xml
```

add the following content to mappers.xml

```xml
<?xml version="1.0" encoding="utf-8" ?>
<!DOCTYPE mapper PUBLIC "-//juice.org//DTD Config 1.0//EN"
        "https://raw.githubusercontent.com/eatmoreapple/juice/main/mapper.dtd">

<mapper namespace="main.Repository">
    <select id="HelloWorld">
        <if test="1 == 1">  <!-- always be true -->
            select "hello world"
        </if>
    </select>
</mapper>
```

```shell
touch main.go
```

add the following content to main.go

```go
package main

import (
	"context"
	"fmt"
	"github.com/eatmoreapple/juice"
	_ "github.com/go-sql-driver/mysql"
)

type Repository interface {
	HelloWorld(ctx context.Context) (string, error)
}

type RepositoryImpl struct{}

func (r RepositoryImpl) HelloWorld(ctx context.Context) (string, error) {
	manager := juice.ManagerFromContext(ctx)
	var iface Repository = r
	executor := juice.NewGenericManager[string](manager).Object(iface.HelloWorld)
	return executor.QueryContext(ctx, nil)
}

func main() {
	cfg, err := juice.NewXMLConfiguration("config.xml")
	if err != nil {
		panic(err)
	}
	engine, err := juice.Default(cfg)
	if err != nil {
		panic(err)
	}
	ctx := juice.ContextWithManager(context.Background(), engine)
	repo := RepositoryImpl{}
	result, err := repo.HelloWorld(ctx)
	fmt.Println(result, err) // hello world <nil>
}
```

```shell
go run main.go
```

### API Documentation

[English](https://juice-doc.readthedocs.io/projects/juice-doc-en/en/latest/)
[简体中文](https://juice-doc.readthedocs.io/en/latest/index.html)


### License

Juice is licensed under the Apache License, Version 2.0. See LICENSE for the full license text.

## Support Me

If you like my work, please consider supporting me by buying me a coffee.

<a href="https://raw.githubusercontent.com/eatmoreapple/eatmoreapple/main/img/wechat_pay.jpg" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" width="150" ></a>

