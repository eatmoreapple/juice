## Pillow

```shell
touch config.xml
```

add the following to config.xml

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
                <foreach collection="users" item="user" separator=", ">
                    (#{user.name}, #{user.age})
                </foreach>
            </insert>

        </mapper>
    </mappers>
</configuration>


```

```golang
package main

import (
	"fmt"
	"github.com/eatmoreapple/pillow"
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

var instance UserRepository = &userRepository{}

func NewUserRepository(engine *pillow.Engine) UserRepository {
	return &userRepository{engine}
}

type userRepository struct {
	engine *pillow.Engine
}

func (u *userRepository) Count() (int64, error) {
	return pillow.NewGenericEngine[int64, any](u.engine).Statement(instance.Count).Query(nil).One()
}

func (u *userRepository) GetUserByID(user *User) (*User, error) {
	return pillow.NewGenericEngine[*User, *User](u.engine).Statement(instance.GetUserByID).Query(user).One()
}

func (u *userRepository) UpdateUser(user *User) (int64, error) {
	result, err := pillow.NewGenericEngine[int64, *User](u.engine).Statement(instance.UpdateUser).Exec(user)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (u *userRepository) DeleteUserByID(user *User) (int64, error) {
	result, err := pillow.NewGenericEngine[int64, *User](u.engine).Statement(instance.DeleteUserByID).Exec(user)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (u *userRepository) CreateUser(user *User) (int64, error) {
	result, err := pillow.NewGenericEngine[int64, *User](u.engine).Statement(instance.CreateUser).Exec(user)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (u *userRepository) BatchCreateUser(users []*User) (int64, error) {
	param := map[string][]*User{
		"users": users,
	}
	result, err := pillow.NewGenericEngine[int64, map[string][]*User](u.engine).Statement(instance.BatchCreateUser).Exec(param)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func main() {

	cfg, err := pillow.NewXMLConfiguration("config.xml")
	if err != nil {
		panic(err)
	}

	engine, err := pillow.DefaultEngine(cfg)
	if err != nil {
		panic(err)
	}

	var repo = NewUserRepository(engine)

	var user = User{
		Id:   1,
		Name: "test",
		Age:  18,
	}

	fmt.Println(repo.CreateUser(&user))

	fmt.Println(repo.Count())

	fmt.Println(repo.GetUserByID(&user))

	fmt.Println(repo.UpdateUser(&user))

	fmt.Println(repo.DeleteUserByID(&user))

	user.Age++

	fmt.Println(repo.BatchCreateUser([]*User{&user}))

}

```