package gorestful

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"reflect"
)

type Page struct {
	Search string `form:"search"`
	Offset int    `form:"offset"`
	Limit  int    `form:"limit"`
}

func processReflectList(res *Resource, list interface{}) []map[string]interface{} {
	// 遍历进行替换处理
	var list1 []map[string]interface{}
	s := reflect.ValueOf(list).Elem()
	for i := 0; i < s.Len(); i++ {
		m := make(map[string]interface{})
		for _, f := range res.Fields {
			v := s.Index(i).FieldByName(f.Name)
			// 存在空值的情况
			if !v.IsValid() {
				m[f.JsonName] = nil
				continue
			}
			if v.Type().Kind() == reflect.Struct {
				switch v.Type().String() {
				case "sql.NullTime":
					if v.FieldByName("Valid").Bool() {
						m[f.JsonName] = v.FieldByName("Time").Interface()
					} else {
						m[f.JsonName] = ""
					}
				case "gorm.Model":
				default:
					m[f.JsonName] = v.Interface()
				}
			} else {
				m[f.JsonName] = v.Interface()
			}
		}
		list1 = append(list1, m)
	}
	return list1
}

// defaultQuery 默认查询所有字段
func defaultQuery(keyword string, q *gorm.DB, res *Resource) *gorm.DB {
	query := ""
	var querySearch []interface{}

	res.EachStringField(func(f Field) {
		if len(query) > 0 {
			query += " or "
		}
		query += f.JsonName + " like ? "
		querySearch = append(querySearch, "%"+keyword+"%")
	})

	if len(querySearch) == 0 {
		return q
	}
	return q.Where(query, querySearch...)
}

// AddResourceApiToGin 插入到gin的路由中去，形成api
// name 资源的名称，比如user
// r gin的group对象，比如绑定了/api/v1
func AddResourceApiToGin(res *Resource) {
	// 列表
	res.apiRouterGroup.GET("/"+res.Name, func(c *gin.Context) {
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
		list := reflect.New(reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(res.getModel()).Elem()), 0, 0).Type()).Interface()
		q := res.getDb(c).Model(res.getModel())
		if len(page.Search) > 0 {
			if res.queryFn != nil {
				q = res.queryFn(page.Search, q, res)
			} else {
				q = defaultQuery(page.Search, q, res)
			}
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

		err = q.Order(res.keyId + " desc").Limit(page.Limit).Offset(page.Offset).Find(list).Error
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
				"list":  processReflectList(res, list),
			},
		})
	})

	// 新增
	res.apiRouterGroup.POST("/"+res.Name, func(c *gin.Context) {
		// 解析
		model := res.getModel()
		err := c.ShouldBindJSON(model)
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "add failed:" + err.Error(),
			})
			return
		}

		// 插入
		err = res.getDb(c).Model(res.getModel()).Create(model).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "add failed:" + err.Error(),
			})
			return
		}

		id := uint(0)
		v := reflect.ValueOf(model).Elem()
		typeOfS := v.Type()
		for i := 0; i < typeOfS.NumField(); i++ {
			log.Println(typeOfS.Field(i).Name)

			if typeOfS.Field(i).Type.Kind() == reflect.Struct {
				if typeOfS.Field(i).Type.String() == "gorm.Model" {
					id = v.Field(i).FieldByName("ID").Interface().(uint)
					break
				}
			}

			if typeOfS.Field(i).Name == "ID" {
				id = v.Field(i).Interface().(uint)
				break
			}
		}

		if res.afterInsert != nil {
			if err = res.afterInsert(c, uint(id)); err != nil {
				c.JSON(200, gin.H{
					"code":    500,
					"message": "add failed:" + err.Error(),
				})
				return
			}
		}

		c.JSON(200, gin.H{
			"code": 200,
			"data": id,
		})
	})

	// 删除
	res.apiRouterGroup.DELETE("/"+res.Name+"/:id", func(c *gin.Context) {
		// 查找
		model := res.getModel()
		err := res.getDb(c).Model(model).Find(model, "id=?", c.Param("id")).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "not found:" + err.Error(),
			})
			return
		}

		err = res.getDb(c).Model(model).Delete("id=?", c.Param("id")).Error
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
	res.apiRouterGroup.GET("/"+res.Name+"/:id", func(c *gin.Context) {
		// 查找
		model := res.getModel()
		err := res.getDb(c).Model(model).Find(model, "id=?", c.Param("id")).Error
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
	res.apiRouterGroup.POST("/"+res.Name+"/:id", func(c *gin.Context) {
		// 解析
		model := res.getModel()
		err := res.getDb(c).Model(model).Find(model, "id=?", c.Param("id")).Error
		if err != nil {
			c.JSON(200, gin.H{
				"code":    500,
				"message": "not found:" + err.Error(),
			})
			return
		}

		modelPost := res.getModel()
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
		err = res.getDb(c).Model(model).Select("*").Omit(res.OmitFields()...).Updates(modelPost).Error
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
