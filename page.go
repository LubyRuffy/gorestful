package gorestful

import (
	"embed"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

//go:embed templates/* static/*
var FS embed.FS

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
		})
	})

	return nil
}
