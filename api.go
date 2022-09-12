package gorestful

import (
	"reflect"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Resource 资源对象
type Resource struct {
	Name   string // 名称
	Fields string // 字段，*或者空表示所有
}

// AddResourceToGin 插入到gin的路由中去，形成api
// name 资源的名称，比如user
// r gin的group对象，比如绑定了/api/v1
func AddResourceToGin(res *Resource, r *gin.RouterGroup, getDb func() *gorm.DB, getModel func() interface{}) {
	// 列表
	r.GET("/"+res.Name, func(c *gin.Context) {
		var count int64
		err := getDb().Model(getModel()).Count(&count).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "list failed:" + err.Error(),
			})
			return
		}

		// 相当于： &[]User
		list := reflect.New(reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(getModel()).Elem()), 0, 0).Type()).Interface()
		err = getDb().Model(getModel()).Find(list).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "list failed:" + err.Error(),
			})
			return
		}
		c.JSON(200, gin.H{
			"code": 200,
			"data": gin.H{
				"count": count,
				"list":  list,
			},
		})
	})

	// 新增
	r.POST("/"+res.Name, func(c *gin.Context) {
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

		id := int64(0)
		v := reflect.ValueOf(model).Elem()
		typeOfS := v.Type()
		for i := 0; i < typeOfS.NumField(); i++ {
			if typeOfS.Field(i).Name == "ID" {
				id = v.Field(i).Int()
			}

		}

		c.JSON(200, gin.H{
			"code": 200,
			"data": id,
		})
	})

	// 删除
	r.DELETE("/"+res.Name+"/:id", func(c *gin.Context) {
		// 查找
		model := getModel()
		err := getDb().Model(model).Find(model, "id=?", c.Param("id")).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "not found:" + err.Error(),
			})
			return
		}

		err = getDb().Model(model).Delete("id=?", c.Param("id")).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "delete failed:" + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"code": 200,
			"data": true,
		})
	})

	// 查看
	r.GET("/"+res.Name+"/:id", func(c *gin.Context) {
		// 查找
		model := getModel()
		err := getDb().Model(model).Find(model, "id=?", c.Param("id")).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "not found:" + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"code": 200,
			"data": model,
		})
	})
}
