package main

import (
	"fmt"
	"github.com/LubyRuffy/gorestful"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
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
		Name:   "user",
		Fields: "*",
	}

	g := gin.Default()
	temp := template.Must(template.New("").Delims("{{{", "}}}").ParseFS(gorestful.FS, "templates/*.html"))
	g.SetHTMLTemplate(temp)

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
