package main

import (
	"github.com/LubyRuffy/gorestful"
	"github.com/LubyRuffy/gorestful/examples/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
)

type ProxyLevel int

const (
	ProxyLevelUnknown ProxyLevel = iota // 未知
	ProxyLevelFirst
	ProxyLevelSecond
	ProxyLevelThird
)

type Article struct {
	ID    uint
	Title string
	Level ProxyLevel // 自定义级别
}

func fillData(gdb *gorm.DB) {
	var count int64
	if gdb.Model(&Article{}).Count(&count).Error == nil && count == 0 {
		gdb.Create(&Article{
			Title: "test",
			Level: ProxyLevelFirst,
		})
	}
}

func main() {
	gdb, err := utils.InitDb(&Article{})
	if err != nil {
		panic(err)
	}

	// 确保生成一个数据用于观察
	fillData(gdb)

	g := gin.Default()

	res, err := gorestful.NewResource(
		gorestful.WithGinEngine(g),
		gorestful.WithGormDb(func(c *gin.Context) *gorm.DB {
			return gdb
		}),
		gorestful.WithUserStruct(func() interface{} {
			return &Article{}
		}),
	)
	if err != nil {
		panic(err)
	}

	// 替换字段
	res.SetEnumField("Level", [][]interface{}{
		{ProxyLevelUnknown, "ProxyLevelUnknown"},
		{ProxyLevelFirst, "ProxyLevelFirst"},
		{ProxyLevelSecond, "ProxyLevelSecond"},
		{ProxyLevelThird, "ProxyLevelThird"},
	})

	gorestful.AddResourceApiPageToGin(res)

	// 默认页面跳转
	g.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, res.PageUrl())
	})
	g.Run(":8080")
}
