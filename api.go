package gorestful

import (
	"reflect"

	"github.com/gin-gonic/gin"
)

type Page struct {
	Search string `form:"search"`
	Offset int    `form:"offset"`
	Limit  int    `form:"limit"`
}

// AddResourceApiToGin 插入到gin的路由中去，形成api
// name 资源的名称，比如user
// r gin的group对象，比如绑定了/api/v1
func AddResourceApiToGin(res *Resource) {
	// 没有绑定
	if res.ApiRouterGroup == nil {
		return
	}

	// 列表
	res.ApiRouterGroup.GET("/"+res.Name, func(c *gin.Context) {
		var page Page
		if err := c.ShouldBindQuery(&page); err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "list failed:" + err.Error(),
			})
			return
		}
		if page.Limit == 0 {
			page.Limit = 10
		}

		// 相当于： &[]User
		list := reflect.New(reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(res.GetModel()).Elem()), 0, 0).Type()).Interface()
		q := res.GetDb().Model(res.GetModel())
		if len(page.Search) > 0 {
			query := ""
			var querySearch []interface{}
			for _, f := range res.Fields {
				if f.CloseEdit {
					continue
				}
				if len(query) > 0 {
					query += " or "
				}
				query += f.Name + " like ? "
				querySearch = append(querySearch, "%"+page.Search+"%")
			}
			q = q.Where(query, querySearch...)
		}

		var count int64
		err := q.Count(&count).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "list failed:" + err.Error(),
			})
			return
		}

		err = q.Order("id desc").Limit(page.Limit).Offset(page.Offset).Find(list).Error
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
	res.ApiRouterGroup.POST("/"+res.Name, func(c *gin.Context) {
		// 解析
		model := res.GetModel()
		err := c.ShouldBindJSON(model)
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "add failed:" + err.Error(),
			})
			return
		}

		// 插入
		err = res.GetDb().Model(res.GetModel()).Save(model).Error
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
	res.ApiRouterGroup.DELETE("/"+res.Name+"/:id", func(c *gin.Context) {
		// 查找
		model := res.GetModel()
		err := res.GetDb().Model(model).Find(model, "id=?", c.Param("id")).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "not found:" + err.Error(),
			})
			return
		}

		err = res.GetDb().Model(model).Delete("id=?", c.Param("id")).Error
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
	res.ApiRouterGroup.GET("/"+res.Name+"/:id", func(c *gin.Context) {
		// 查找
		model := res.GetModel()
		err := res.GetDb().Model(model).Find(model, "id=?", c.Param("id")).Error
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

	// 修改
	res.ApiRouterGroup.POST("/"+res.Name+"/:id", func(c *gin.Context) {
		// 解析
		model := res.GetModel()
		err := res.GetDb().Model(model).Find(model, "id=?", c.Param("id")).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "not found:" + err.Error(),
			})
			return
		}

		modelPost := res.GetModel()
		err = c.ShouldBindJSON(modelPost)
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "edit failed:" + err.Error(),
			})
			return
		}

		// 删除closeedit的列
		unsetField(res, modelPost)

		// 修改
		err = res.GetDb().Model(model).Updates(modelPost).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "add failed:" + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"code": 200,
			"data": reflect.ValueOf(model).Elem().FieldByName("ID").Uint(),
		})
	})
}
