package gorestful

import (
	"html/template"
	"io/fs"
	"net/http"
	"reflect"
)

// AddResourceApiPageToGin 增加api和page页面到gin路由
func AddResourceApiPageToGin(res ...*Resource) {
	for _, r := range res {
		AddResourceApiToGin(r)
		AddResourcePageToGin(r)
	}
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

// loadFS 加载内嵌的模板
func loadFS(res *Resource) {
	temp := template.Must(template.New("").Delims("{{{", "}}}").Funcs(map[string]interface{}{
		"toJS": func(s string) template.JS {
			return template.JS(s)
		},
	}).ParseFS(FS, "templates/*.html"))
	res.ginEngine.SetHTMLTemplate(temp)
	fsys, err := fs.Sub(FS, "static")
	if err != nil {
		panic(err)
	}
	res.ginEngine.StaticFS("/static", http.FS(fsys))
}
