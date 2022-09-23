package gorestful

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

type AuthMode interface {
	Init() error // 初始化
	URL() string
}

type LoginField struct {
	Name        string
	DisplayName string
	Type        string
}

// EmbedLogin 内嵌的登录
type EmbedLogin struct {
	RouterGroup *gin.RouterGroup                                              // 登录挂载的地址
	Name        string                                                        // 登录的名称，默认是login，可以是其他的
	LoginFields []LoginField                                                  // 登录表单列表
	CheckValid  func(e *EmbedLogin, formMap map[string]string) (string, bool) // 是否有效的账号
	Error       string                                                        // 错误提示
	Key         []byte                                                        // key的内容，用于jwt加密
}

// defaultLoginField 默认登录表单项
var (
	defaultLoginField = []LoginField{
		{
			Name:        "user",
			DisplayName: "User",
			Type:        "text",
		},
		{
			Name:        "pass",
			DisplayName: "Password",
			Type:        "password",
		},
	}
	defaultJwtKey = "%gorestful%for%everyone%who%need%"
)

// Init 初始化
func (e *EmbedLogin) Init() error {
	if e.RouterGroup == nil {
		return errors.New("AuthMiddle has no RouterGroup")
	}
	if e.Name == "" {
		e.Name = "login"
	}
	if e.LoginFields == nil {
		e.LoginFields = defaultLoginField
	}
	if e.Key == nil {
		e.Key = []byte(defaultJwtKey)
	}

	// 登录页面
	e.RouterGroup.GET("/"+e.Name, func(c *gin.Context) {
		if referer := c.Request.Referer(); referer != "" {
			//c.Set("referer", referer)
			c.SetCookie("referer", referer, 60, "/", "", false, true)
		}
		c.HTML(http.StatusOK, "login.html", e)
	})
	e.RouterGroup.POST("/"+e.Name, func(c *gin.Context) {
		formMap := make(map[string]string)
		for _, f := range e.LoginFields {
			formMap[f.Name], _ = c.GetPostForm(f.Name)
		}
		if token, ok := e.CheckValid(e, formMap); ok {
			var referer string
			if v, err := c.Cookie("referer"); err == nil {
				referer = v
			}
			c.HTML(http.StatusOK, "redirect.html", gin.H{
				"Token":   token,
				"Referer": referer,
			})
		} else {
			e.Error = "not valid"
			c.HTML(http.StatusOK, "login.html", e)
			e.Error = ""
		}
	})
	logout := func(c *gin.Context) {
		//c.Set("referer", "")
		// 设置cookie  MaxAge设置为-1，表示删除cookie
		c.SetCookie("referer", "", -1, "/", "", false, true)
		c.HTML(http.StatusOK, "login.html", e)
		//todo:如何作废已有的jwt token？
	}
	// 退出
	e.RouterGroup.DELETE("/"+e.Name, logout)
	e.RouterGroup.Any("/logout", logout)
	return nil
}

// URL 跳转的地址
func (e EmbedLogin) URL() string {
	base := e.RouterGroup.BasePath()
	if base == "/" {
		base = ""
	}
	return base + "/" + e.Name
}

type AuthMiddle struct {
	URL               string   // 跳转的地址
	HeaderKey         string   // auth的header头对应的key，默认是Authorization，可以自行修改
	HeaderValuePrefix string   // auth的header头对应的value前缀，比如Token
	AuthMode          AuthMode // 是否内嵌登录
}

// AddAuthToGin 增加认证
func AddAuthToGin(am *AuthMiddle) {
	if err := am.AuthMode.Init(); err != nil {
		panic(err)
	}
	if am.URL == "" {
		am.URL = am.AuthMode.URL()
	}
	if am.HeaderKey == "" {
		am.HeaderKey = "Authorization"
	}
	if am.HeaderValuePrefix == "" {
		am.HeaderValuePrefix = "Token "
	} else if am.HeaderValuePrefix == "-" {
		am.HeaderValuePrefix = ""
	}
}