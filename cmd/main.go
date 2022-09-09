package main

import (
	"fmt"
	"github.com/LubyRuffy/gorestful"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

	g := gin.Default()
	v1 := g.Group("/api/v1")
	gorestful.AddResourceToGinRouter("user", v1, func() *gorm.DB {
		return gdb
	}, func() interface{} {
		return &User{}
	})

	g.Run(":9999")
}
