package main

import (
	"fmt"
	"github.com/LubyRuffy/gorestful"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v4"
	"github.com/mitchellh/mapstructure"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/http"
	"strings"
)

type Article struct {
	ID     uint
	Title  string
	UserId int
	//User   *User
}

type User struct {
	ID       uint
	UserName string
	Password string
}

func initDb() (*gorm.DB, error) {
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	dsn := fmt.Sprintf("%s?journal_mode=%s&busy_timeout=%s", "b.sqlite", "WAL", "9999999")
	gdb, err := gorm.Open(sqlite.Open(dsn), cfg)
	if err != nil {
		return nil, err
	}

	return gdb, gdb.AutoMigrate(&Article{}, &User{})
}

func main() {
	gdb, err := initDb()
	if err != nil {
		panic(err)
	}

	// 插入两条记录
	gdb.Create(&User{
		ID:       1,
		UserName: "user1",
		Password: "pass1",
	})
	gdb.Create(&User{
		ID:       2,
		UserName: "user2",
		Password: "pass2",
	})
	gdb.Create(&Article{
		ID:     1,
		Title:  "a1",
		UserId: 1,
	})
	gdb.Create(&Article{
		ID:     2,
		Title:  "a2",
		UserId: 2,
	})

	g := gin.Default()

	// jwt加密
	type MyClaims struct {
		jwt.RegisteredClaims
		UID      uint
		Username string
	}

	var userInfo func(c *gin.Context) *MyClaims
	am := &gorestful.AuthMiddle{
		GetUser: func(c *gin.Context) string {
			if info := userInfo(c); info != nil {
				return info.Username
			}
			return ""
		},
		AuthMode: &gorestful.EmbedLogin{
			RouterGroup: g.Group("/"), // /login
			LoginFields: []*gorestful.LoginField{{Name: "username"}, {Name: "password"}},
			CheckLogin: func(c *gin.Context, e *gorestful.EmbedLogin, formMap map[string]string) (string, bool) {
				var checkUser User
				if err := mapstructure.Decode(formMap, &checkUser); err != nil {
					return "", false
				}
				var user User
				if err = gdb.Where(&checkUser).Find(&user).Error; err == nil && user.ID > 0 {
					log.Println(user.UserName, "auth ok")
					t := jwt.NewWithClaims(jwt.SigningMethodHS512, MyClaims{
						UID:      user.ID,
						Username: user.UserName,
					})

					if tokenString, err := t.SignedString(e.Key); err == nil {
						return tokenString, true
					} else {
						log.Println("jwt failed:", err)
					}
					return "", false
				}
				return "", false
			},
		},
	}

	userInfo = func(c *gin.Context) *MyClaims {
		if tokenString := c.Request.Header.Get(am.HeaderKey); tokenString != "" {
			if len(am.HeaderValuePrefix) > 0 && strings.Contains(tokenString, am.HeaderValuePrefix) {
				tokenString = strings.Split(tokenString, am.HeaderValuePrefix)[1]
			}
			token, err := jwt.ParseWithClaims(tokenString, &MyClaims{}, func(token *jwt.Token) (interface{}, error) {
				// Don't forget to validate the alg is what you expect:
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}

				// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
				return am.AuthMode.(*gorestful.EmbedLogin).Key, nil
			})
			if err == nil {
				if claims, ok := token.Claims.(*MyClaims); ok && token.Valid {
					log.Println(claims.Username)
					c.Set("user_id", claims.UID)
					c.Set("user_name", claims.Username)
					return claims
				}
			}
			log.Println("login failed:", err)
		}
		return nil
	}

	apiGroup := g.Group("/api", func(c *gin.Context) {
		if claims := userInfo(c); claims != nil {
			log.Println(claims.Username)
			c.Set("user_id", claims.UID)
			c.Set("user_name", claims.Username)
			return
		}

		c.AbortWithStatusJSON(403, map[string]interface{}{
			"code":    403,
			"message": "invalid auth",
		})
	})

	res, err := gorestful.NewResource(
		gorestful.WithGinEngine(g),
		gorestful.WithGormDb(func(c *gin.Context) *gorm.DB {
			v, exists := c.Get("user_id")
			if !exists {
				panic("not valid user")
			}
			if uid := v.(uint); uid > 0 {
				return gdb.Model(&Article{}).Where("user_id=?", uid) // 只显示当前用户的资源
			}
			panic("not valid user")
		}),
		gorestful.WithUserStruct(func() interface{} {
			return &Article{}
		}),
		gorestful.WithAuthMiddle(am),
		gorestful.WithApiRouterGroup(apiGroup),
	)
	if err != nil {
		panic(err)
	}

	gorestful.AddResourceApiPageToGin(res)

	// 默认页面跳转
	g.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, res.PageUrl())
	})
	g.Run(":8080")
}
