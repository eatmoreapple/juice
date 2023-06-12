## Juice SQL Mapper Framework For Golang

![Go Doc](https://pkg.go.dev/badge/github.com/eatmoreapple/juice)
![Go Report Card](https://goreportcard.com/badge/github.com/eatmoreapple/juice)
![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

This is a SQL mapper framework for Golang. It is inspired by MyBatis.

Juice is a simple and lightweight framework. It is easy to use and easy to extend.

### Example

```shell
vim config.xml
```

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
vim mappers.xml
```

```xml
<?xml version="1.0" encoding="utf-8" ?>
<!DOCTYPE mapper PUBLIC "-//mybatis.org//DTD Mapper 3.0//EN"
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
vim main.go
```

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
	engine, err := juice.DefaultEngine(cfg)
	if err != nil {
		panic(err)
	}
	ctx := juice.ContextWithManager(context.Background(), engine)
	repo := RepositoryImpl{}
	result, err := repo.HelloWorld(ctx)
	fmt.Println(result, err) // hello world <nil>
}
```


### Document

[Read the document](https://juice-doc.readthedocs.io/en/latest/index.html)

### License

Juice is licensed under the Apache License, Version 2.0. See LICENSE for the full license text.

### Buy me a cup of coffee

<img width="210px"  src="https://github.com/eatmoreapple/eatMoreApple/blob/main/img/wechat_pay.jpg" align="left">
