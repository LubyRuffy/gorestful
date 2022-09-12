package gorestful

import (
	"embed"
	"github.com/gin-gonic/gin"
	"net/http"
)

//go:embed templates/*
var FS embed.FS

// AddResourcePageToGin 生成页面
func AddResourcePageToGin(res *Resource, r *gin.RouterGroup, apiGroup *gin.RouterGroup) {
	r.GET("/"+res.Name, func(c *gin.Context) {
		c.HTML(http.StatusOK, "resource.html", gin.H{
			"resource":  res,
			"apiPrefix": apiGroup.BasePath(),
		})
	})

	r.GET("/"+res.Name+"_new", func(c *gin.Context) {
		c.HTML(http.StatusOK, "resource_new.html", gin.H{
			"resource":  res,
			"apiPrefix": apiGroup.BasePath(),
		})
	})
}
