package gorestful

import (
	"errors"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"net/http"
	"reflect"
)

type AuthMode interface {
	Init() error // 初始化
	URL() string
	GetRouterGroup() *gin.RouterGroup
}

type LoginField struct {
	Name        string
	DisplayName string
	Type        string
}

// EmbedLogin 内嵌的登录
type EmbedLogin struct {
	RouterGroup  *gin.RouterGroup                                                              // 登录挂载的地址
	Name         string                                                                        // 登录的名称，默认是login，可以是其他的
	LoginFields  []*LoginField                                                                 // 登录表单列表
	CheckLogin   func(c *gin.Context, e *EmbedLogin, formMap map[string]string) (string, bool) // 是否有效的账号，返回值为token和是否有效，只在登录的时候有用
	Key          []byte                                                                        // key的内容，用于jwt加密
	OpenRegister bool                                                                          // 是否开放注册
	Register     func(c *gin.Context, e *EmbedLogin, formMap map[string]string) error          // 是否注册成功
}

// defaultLoginField 默认登录表单项
var (
	defaultLoginField = []*LoginField{
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

// loginForm 获取登录表单提交的信息，转成map模式
func (e *EmbedLogin) loginForm(c *gin.Context) map[string]string {
	formMap := make(map[string]string)
	for _, f := range e.LoginFields {
		formMap[f.Name], _ = c.GetPostForm(f.Name)
	}
	return formMap
}

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
	for _, f := range e.LoginFields {
		if f.Type == "" {
			f.Type = "text"
		}
		if f.DisplayName == "" {
			f.DisplayName = cases.Title(language.AmericanEnglish).String(f.Name)
		}
	}
	if e.Key == nil {
		e.Key = []byte(defaultJwtKey)
	}

	// 登录页面
	e.RouterGroup.GET("/"+e.Name, func(c *gin.Context) {
		if referer := c.Request.Referer(); referer != "" {
			if c.Request.RequestURI != e.URL() && c.Request.RequestURI != "/logout" {
				//c.Set("referer", referer)
				c.SetCookie("referer", referer, 60, "/", "", false, true)
			}
		}
		c.HTML(http.StatusOK, "login.html", gin.H{
			"title": "Login",
			"auth":  e,
		})
	})
	e.RouterGroup.POST("/"+e.Name, func(c *gin.Context) {
		formMap := e.loginForm(c)
		if token, ok := e.CheckLogin(c, e, formMap); ok {
			var referer string
			if v, err := c.Cookie("referer"); err == nil {
				referer = v
			} else {
				if len(RegisteredResourcesPage) > 0 {
					referer = reflect.ValueOf(RegisteredResourcesPage).MapKeys()[0].String()
				} else {
					referer = "/"
				}

			}
			c.HTML(http.StatusOK, "redirect.html", gin.H{
				"Token":   token,
				"Referer": referer,
			})
		} else {
			c.HTML(http.StatusOK, "login.html", gin.H{
				"title": "Login",
				"auth":  e,
				"error": "not valid",
			})
		}
	})
	logout := func(c *gin.Context) {
		//c.Set("referer", "")
		// 设置cookie  MaxAge设置为-1，表示删除cookie
		c.SetCookie("referer", "", -1, "/", "", false, true)
		c.HTML(http.StatusOK, "login.html", gin.H{
			"title": "Login",
			"auth":  e,
		})
		//todo:如何作废已有的jwt token？
	}
	// 退出
	e.RouterGroup.DELETE("/"+e.Name, logout)
	e.RouterGroup.Any("/logout", logout)

	// 注册用户
	if e.OpenRegister {
		if e.Register == nil {
			panic("should set Register callback function when set OpenRegister")
		}
		e.RouterGroup.GET("/register", func(c *gin.Context) {
			c.HTML(http.StatusOK, "register.html", gin.H{
				"title": "Register",
				"auth":  e,
			})
		})
		e.RouterGroup.POST("/register", func(c *gin.Context) {
			formMap := e.loginForm(c)
			err := e.Register(c, e, formMap)
			if err == nil {
				c.Redirect(http.StatusFound, "/login")
				return
			}
			c.HTML(http.StatusOK, "register.html", gin.H{
				"title": "Register",
				"auth":  e,
				"error": err.Error(),
			})
		})
	}

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

// GetRouterGroup 绑定的地址
func (e EmbedLogin) GetRouterGroup() *gin.RouterGroup {
	return e.RouterGroup
}

type AuthMiddle struct {
	URL               string                      // 跳转的地址，可以不填，就由AuthMode.URL来获取
	HeaderKey         string                      // auth的header头对应的key，默认是Authorization，可以自行修改
	HeaderValuePrefix string                      // auth的header头对应的value前缀，比如Token
	AuthMode          AuthMode                    // 是否内嵌登录
	GetUser           func(c *gin.Context) string // 获取当前登录的ID
}

// addAuthToGin 增加认证
func addAuthToGin(am *AuthMiddle) {
	if err := am.AuthMode.Init(); err != nil {
		panic(err)
	}

	// 获取用户信息
	am.AuthMode.GetRouterGroup().GET("/userinfo", func(c *gin.Context) {
		if am.GetUser != nil {
			c.JSON(http.StatusOK, gin.H{
				"code": 200,
				"data": gin.H{
					"username": am.GetUser(c),
				},
			})
			return
		}
		c.JSON(http.StatusOK, nil)
	})

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
