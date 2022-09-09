package gorestful

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Model interface {
	GetID() uint
}

// AddResourceToGinRouter 插入到gin的路由中去
// name 资源的名称，比如user
// r gin的group对象，比如绑定了/api/v1
func AddResourceToGinRouter(name string, r *gin.RouterGroup, getDb func() *gorm.DB, model interface{}) {
	// 列表
	r.GET("/"+name, func(c *gin.Context) {
		var count int64
		err := getDb().Model(getModel()).Count(&count).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "list failed:" + err.Error(),
			})
			return
		}

		var list []model
		c.JSON(200, gin.H{
			"code": 200,
			"data": gin.H{
				"count": count,
				"list":  getDb().Model(&model).Find(&list),
			},
		})
	})

	// 新增
	r.POST("/"+name, func(c *gin.Context) {
		// 解析
		model := getModel()
		err := c.ShouldBindJSON(model)
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "add failed:" + err.Error(),
			})
			return
		}

		// 插入
		err = getDb().Model(getModel()).Save(model).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "add failed:" + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"code": 200,
			"data": model.GetID(),
		})
	})
}
