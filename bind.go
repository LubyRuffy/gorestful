package gorestful

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"html/template"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
)

type Field struct {
	Name      string // 字段名称
	JsonName  string // json的名称，可以跟数据库中的字段名称不一样
	Type      string // 字段类型
	CloseEdit bool   // 是否停止编辑？比如id或者创建时间之类的应该不让编辑
	DenyNull  bool   // 是否不允许空值
	// 格式？
	// 合法性校验？
}

// Resource 资源对象
type Resource struct {
	Name            string           // 名称
	Fields          []Field          // 字段，*或者空表示所有
	BlackFields     []string         // 黑名单字段，不进行显示和编辑
	ApiRouterGroup  *gin.RouterGroup // api绑定的地址
	PageRouterGroup *gin.RouterGroup // page页面绑定的地址
	GetDb           func(c *gin.Context) *gorm.DB
	GetModel        func() interface{}
	AuthMiddle      *AuthMiddle                         // 认证中间件
	ID              string                              // 主键id，默认为id
	AfterInsert     func(c *gin.Context, id uint) error // 插入数据后的通知事件
}

// OmitFields 获取不能编辑的字段
func (res *Resource) OmitFields() []string {
	var fs []string
	for _, f := range res.Fields {
		if f.CloseEdit {
			fs = append(fs, f.Name)
		}
	}
	return fs
}

// addValue 解析一个StructField为field
func (res *Resource) addValue(val reflect.StructField, closeEdit bool) {
	// 是否黑名单
	if res.isBlackField(val.Name) {
		return
	}

	// ProxyLevel => models.ProxyAnonymityLevel => int
	//log.Println(val.Name, "=>", val.Type.String(), "=>", val.Type.Kind().String())

	jsonName := val.Tag.Get("json")
	if jsonName == "" {
		jsonName = val.Name
	}

	var denyNull bool
	gormTag := val.Tag.Get("gorm")
	if strings.Contains(gormTag, "index:") {
		denyNull = true
	}

	res.Fields = append(res.Fields, Field{
		Name:      val.Name,
		Type:      val.Type.Kind().String(), // 不能用val.Type.String()
		JsonName:  jsonName,
		CloseEdit: closeEdit,
		DenyNull:  denyNull,
	})
}

// isBlackField 是否黑名单
func (res *Resource) isBlackField(name string) bool {
	for _, blackField := range res.BlackFields {
		if blackField == name {
			return true
		}
	}
	return false
}

// autoFill 自动填充字段
func (res *Resource) autoFill() {
	v := reflect.ValueOf(res.GetModel()).Elem()
	typeOfS := v.Type()
	for i := 0; i < typeOfS.NumField(); i++ {
		if typeOfS.Field(i).Type.Kind() == reflect.Struct {
			//log.Println(typeOfS.Field(i).Type.String()) sql.NullTime
			//log.Println(typeOfS.Field(i).Type.Name()) NullTime
			if typeOfS.Field(i).Type.String() == "sql.NullTime" {
				//res.addValue(typeOfS.Field(i), true)
				// 暂时不支持sql.NullTime
				//jsonName := typeOfS.Field(i).Tag.Get("json")
				//if jsonName == "" {
				//	jsonName = typeOfS.Field(i).Name
				//}
				//res.Fields = append(res.Fields, Field{
				//	Name:      typeOfS.Field(i).Name,
				//	Type:      typeOfS.Field(i).Type.Name(),
				//	JsonName:  typeOfS.Field(i).Tag.Get("json"),
				//	CloseEdit: false,
				//})
				continue
			}

			// 结构, gorm.model / sql.NullTime
			for j := 0; j < v.Field(i).Type().NumField(); j++ {
				val := v.Field(i).Type().Field(j)
				if "DeletedAt" == val.Name {
					continue
				}

				res.addValue(val, true)
			}
		} else {
			res.addValue(typeOfS.Field(i), false)
		}
	}
}

// AddResourceApiPageToGin 增加api和page页面到gin路由
func AddResourceApiPageToGin(res *Resource) error {
	if res.Fields == nil {
		// 自动提取
		res.autoFill()
	}

	AddResourceApiToGin(res)
	return AddResourcePageToGin(res)
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
				if !f.CloseEdit {
					continue
				}

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
				if !f.CloseEdit {
					continue
				}

				fName := typeOfS.Field(i).Name
				if fName == f.Name {
					unsetByType(f.Type, v.Field(i))
				}
			}
		}
	}
}

// LoadFS 加载内嵌的模板
func LoadFS(g *gin.Engine) {
	temp := template.Must(template.New("").Delims("{{{", "}}}").Funcs(map[string]interface{}{
		"toJS": func(s string) template.JS {
			return template.JS(s)
		},
	}).ParseFS(FS, "templates/*.html"))
	g.SetHTMLTemplate(temp)
	fsys, err := fs.Sub(FS, "static")
	if err != nil {
		panic(err)
	}
	g.StaticFS("/static", http.FS(fsys))
}
