package main

import (
	"fmt"
	"github.com/LubyRuffy/gorestful"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http"
)

type Article struct {
	ID    uint
	Title string
}

func initDb() (*gorm.DB, error) {
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	dsn := fmt.Sprintf("%s?journal_mode=%s&busy_timeout=%s", "a.sqlite", "WAL", "9999999")
	gdb, err := gorm.Open(sqlite.Open(dsn), cfg)
	if err != nil {
		return nil, err
	}

	return gdb, gdb.AutoMigrate(&Article{})
}

func main() {
	gdb, err := initDb()
	if err != nil {
		panic(err)
	}

	g := gin.Default()

	res, err := gorestful.NewResource(
		gorestful.WithGinEngine(g),
		gorestful.WithGormDb(func(c *gin.Context) *gorm.DB {
			return gdb
		}),
		gorestful.WithUserStruct(func() interface{} {
			return &Article{}
		}),
		gorestful.WithAuthMiddle(&gorestful.AuthMiddle{
			AuthMode: &gorestful.EmbedLogin{
				RouterGroup: g.Group("/"), // /login
				LoginFields: []*gorestful.LoginField{{Name: "token"}},
				CheckLogin: func(c *gin.Context, e *gorestful.EmbedLogin, formMap map[string]string) (string, bool) {
					if token, ok := formMap["token"]; ok && token == "123456" {
						return token, true // 这里可以用jwt的方式进行数据绑定
					}
					return "", false
				},
			},
		}),
	)
	if err != nil {
		panic(err)
	}

	gorestful.AddResourceApiPageToGin(res)

	// 默认页面跳转
	g.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, res.PageUrl())
	})
	g.Run(":8080")
}
