package gorestful

import (
	"embed"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

//go:embed templates/* static/*
var FS embed.FS
var RegisteredResourcesPage = make(map[string]*Resource) // 注册成功的资源

// AddResourcePageToGin 生成页面
func AddResourcePageToGin(res *Resource) error {
	if res.PageRouterGroup == nil {
		// 没有绑定
		return nil
	}

	if res.ApiRouterGroup == nil {
		return errors.New("must bind api router before generate page")
	}

	res.PageRouterGroup.GET("/"+res.Name, func(c *gin.Context) {
		c.HTML(http.StatusOK, "resource.html", gin.H{
			"resource":  res,
			"apiPrefix": res.ApiRouterGroup.BasePath(),
			"title":     res.Name + "list",
		})
	})

	base := res.PageRouterGroup.BasePath()
	if base == "/" {
		base = ""
	}
	pageUri := base + "/" + res.Name

	RegisteredResourcesPage[pageUri] = res

	return nil
}
