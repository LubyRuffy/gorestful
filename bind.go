package gorestful

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"reflect"
)

type Field struct {
	Name      string // 字段名称
	Type      string // 字段类型
	CloseEdit bool   // 是否停止编辑？比如id或者创建时间之类的应该不让编辑
	// 格式？
	// 合法性校验？
}

// Resource 资源对象
type Resource struct {
	Name            string           // 名称
	Fields          []Field          // 字段，*或者空表示所有
	ApiRouterGroup  *gin.RouterGroup // api绑定的地址
	PageRouterGroup *gin.RouterGroup // page页面绑定的地址
	GetDb           func() *gorm.DB
	GetModel        func() interface{}
}

// unsetByType 根据res字段的类型来充值初始值
func unsetByType(fieldType string, v reflect.Value) {
	switch fieldType {
	case "string":
		v.Set(reflect.ValueOf(""))
	case "int":
		v.Set(reflect.ValueOf(0))
	case "uint":
		v.Set(reflect.ValueOf(uint(0)))
	}
}

// unsetFieldValue 递归重置closeedit的列
func unsetFieldValue(res *Resource, v reflect.Value) {
	for j := 0; j < v.Type().NumField(); j++ {
		if v.Field(j).Type().Kind() == reflect.Struct {
			unsetFieldValue(res, v.Field(j))
		} else {
			for _, f := range res.Fields {
				fName := v.Type().Field(j).Name
				if fName == f.Name {
					unsetByType(f.Type, v.Field(j))
				}
			}
		}
	}
}

// unsetField 删除closeedit的列
func unsetField(res *Resource, model interface{}) {
	v := reflect.ValueOf(model).Elem()
	typeOfS := v.Type()
	for i := 0; i < typeOfS.NumField(); i++ {
		if typeOfS.Field(i).Type.Kind() == reflect.Struct {
			unsetFieldValue(res, v.Field(i))
		} else {
			for _, f := range res.Fields {
				fName := typeOfS.Field(i).Name
				if fName == f.Name {
					unsetByType(f.Type, v.Field(i))
				}
			}
		}
	}
}

// AddResourceApiPageToGin 增加api和page页面到gin路由
func AddResourceApiPageToGin(res *Resource) error {
	AddResourceApiToGin(res)
	return AddResourcePageToGin(res)
}
