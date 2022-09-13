package main

import (
	"fmt"
	"github.com/LubyRuffy/gorestful"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"io/fs"
	"net/http"
)

func main() {
	cfg := &gorm.Config{}
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

	res := &gorestful.Resource{
		Name: "user",
		Fields: []gorestful.Field{
			{
				Name:      "ID",
				Type:      "string",
				CloseEdit: true,
			},
			{
				Name: "email",
				Type: "string",
			},
			//{
			//	Name:      "CreatedAt",
			//	Type:      "datetime",
			//	CloseEdit: true,
			//},
		},
	}

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

	apiGroup := g.Group("/api/v1")
	gorestful.AddResourceToGin(res, apiGroup, func() *gorm.DB {
		return gdb
	}, func() interface{} {
		return &User{}
	})

	home := g.Group("/")
	gorestful.AddResourcePageToGin(res, home, apiGroup)

	g.Run(":9999")
}
