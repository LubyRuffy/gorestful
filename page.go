package gorestful

import (
	"embed"
	"github.com/gin-gonic/gin"
	"net/http"
)

//go:embed templates/* static/*
var FS embed.FS
var RegisteredResourcesPage = make(map[string]*Resource) // 注册成功的资源

// AddResourcePageToGin 生成页面
func AddResourcePageToGin(res *Resource) {

	res.pageRouterGroup.GET("/"+res.Name, func(c *gin.Context) {
		c.HTML(http.StatusOK, "resource.html", gin.H{
			"resource":  res,
			"apiPrefix": res.apiRouterGroup.BasePath(),
			"title":     res.Name + "list",
		})
	})

	RegisteredResourcesPage[res.PageUrl()] = res
}
