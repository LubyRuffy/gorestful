package main

import (
	"fmt"
	"github.com/LubyRuffy/gorestful"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"html/template"
	"io/fs"
	"net/http"
)

func main() {
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	dsn := fmt.Sprintf("%s?journal_mode=%s&busy_timeout=%s", "a.sqlite", "WAL", "9999999")
	gdb, err := gorm.Open(sqlite.Open(dsn), cfg)
	if err != nil {
		panic(err)
	}

	type User struct {
		gorm.Model
		Email string `json:"email"`
	}

	err = gdb.AutoMigrate(&User{})

	g := gin.Default()
	temp := template.Must(template.New("").Delims("{{{", "}}}").Funcs(map[string]interface{}{
		"toJS": func(s string) template.JS {
			return template.JS(s)
		},
	}).ParseFS(gorestful.FS, "templates/*.html"))
	g.SetHTMLTemplate(temp)
	fsys, err := fs.Sub(gorestful.FS, "static")
	if err != nil {
		panic(err)
	}
	g.StaticFS("/static", http.FS(fsys))

	res := &gorestful.Resource{
		Name: "user",
		Fields: []gorestful.Field{
			{
				Name:      "ID",
				Type:      "uint",
				CloseEdit: true,
			},
			{
				Name: "email",
				Type: "string",
			},
			{
				Name:      "CreatedAt",
				Type:      "string",
				CloseEdit: true,
			},
		},
		ApiRouterGroup:  g.Group("/api/v1"),
		PageRouterGroup: g.Group("/"),
		GetModel: func() interface{} {
			return &User{}
		},
		GetDb: func() *gorm.DB {
			return gdb
		},
	}

	if err = gorestful.AddResourceApiPageToGin(res); err != nil {
		panic(err)
	}

	g.Run(":9999")
}
