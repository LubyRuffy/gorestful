package gorestful

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	Name string // 名称, 如果为空，根据getModel的结构自动提取

	getDb     func(c *gin.Context) *gorm.DB // 数据库，必须设置
	getModel  func() interface{}            // 模型，必须设置
	ginEngine *gin.Engine                   // gin引擎，必须设置

	Fields          []Field                             // 字段，*或者空表示所有, 如果为空，根据getModel的结构自动提取
	BlackFields     []string                            // 黑名单字段，不进行显示和编辑
	keyId           string                              // 主键id, 如果为空，默认为id
	afterInsert     func(c *gin.Context, id uint) error // 插入数据后的通知事件，默认为空
	authMiddle      *AuthMiddle                         // 认证中间件，留空表示不认证
	apiRouterGroup  *gin.RouterGroup                    // api绑定的地址
	pageRouterGroup *gin.RouterGroup                    // page页面绑定的地址
}

type ResourceOption func(res *Resource) error

// NewResource 创建一个资源
func NewResource(opts ...ResourceOption) (*Resource, error) {
	res := &Resource{}
	for _, f := range opts {
		if err := f(res); err != nil {
			return nil, err
		}
	}

	if res.ginEngine == nil {
		return nil, errors.New("not gin.Engine set, should use WithGinEngine")
	}
	if res.getModel == nil {
		return nil, errors.New("not GetModel set, should use WithUserStruct")
	}

	// <----- 自动提取信息
	if res.apiRouterGroup == nil {
		res.apiRouterGroup = res.ginEngine.Group("/api")
	}
	if res.pageRouterGroup == nil {
		res.pageRouterGroup = res.ginEngine.Group("/")
	}
	if res.keyId == "" {
		res.keyId = "id"
	}
	if res.Name == "" {
		t := reflect.TypeOf(res.getModel())
		//log.Println(t, t.Name(), t.Kind(), t.Elem().Name())
		res.Name = strings.ToLower(t.Elem().Name())
	}
	if res.Fields == nil {
		// 自动提取
		res.autoFill()
	}

	// 加载模板文件
	loadFS(res)

	return res, nil
}

// WithGinEngine 绑定gin.Engine
func WithGinEngine(engine *gin.Engine) ResourceOption {
	return func(res *Resource) error {
		res.ginEngine = engine
		return nil
	}
}

// WithUserStruct 绑定结构体
func WithUserStruct(getModel func() interface{}) ResourceOption {
	return func(res *Resource) error {
		res.getModel = getModel
		return nil
	}
}

// WithName 绑定结构体
func WithName(name string) ResourceOption {
	return func(res *Resource) error {
		res.Name = name
		return nil
	}
}

// WithGormDb 绑定数据库，需要对定义model负责？
func WithGormDb(getDb func(c *gin.Context) *gorm.DB) ResourceOption {
	return func(res *Resource) error {
		res.getDb = getDb
		return nil
	}
}

// WithApiRouterGroup 绑定api到gin的地址
func WithApiRouterGroup(r *gin.RouterGroup) ResourceOption {
	return func(res *Resource) error {
		res.apiRouterGroup = r
		return nil
	}
}

// WithPageRouterGroup 绑定页面到gin的地址
func WithPageRouterGroup(r *gin.RouterGroup) ResourceOption {
	return func(res *Resource) error {
		res.pageRouterGroup = r
		return nil
	}
}

// WithAuthMiddle 绑定页面到gin的地址
func WithAuthMiddle(am *AuthMiddle) ResourceOption {
	return func(res *Resource) error {
		if am != nil {
			addAuthToGin(am)

			res.authMiddle = am
		}

		return nil
	}
}

// WithID 设置id主键的名称
func WithID(id string) ResourceOption {
	return func(res *Resource) error {
		res.keyId = id
		return nil
	}
}

// WithAfterInsert 设置id主键的名称
func WithAfterInsert(f func(c *gin.Context, id uint) error) ResourceOption {
	return func(res *Resource) error {
		res.afterInsert = f
		return nil
	}
}

// AuthMiddle 获取认证中间件，主要用于页面显示
func (res *Resource) AuthMiddle() *AuthMiddle {
	return res.authMiddle
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

// ApiUrl 请求资源的API地址
func (res *Resource) ApiUrl() string {
	base := res.apiRouterGroup.BasePath()
	if base == "/" {
		base = ""
	}
	return base + "/" + res.Name
}

// PageUrl 请求资源的API地址
func (res *Resource) PageUrl() string {
	base := res.pageRouterGroup.BasePath()
	if base == "/" {
		base = ""
	}
	return base + "/" + res.Name
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
	v := reflect.ValueOf(res.getModel()).Elem()
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
