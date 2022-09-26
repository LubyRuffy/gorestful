package main

import (
	"fmt"
	"github.com/LubyRuffy/gorestful"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
)

func main() {
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	dsn := fmt.Sprintf("%s?journal_mode=%s&busy_timeout=%s", "a.sqlite", "WAL", "9999999")
	gdb, err := gorm.Open(sqlite.Open(dsn), cfg)
	if err != nil {
		panic(err)
	}

	type User struct {
		gorm.Model
		Email string `json:"email"`
	}

	err = gdb.AutoMigrate(&User{})

	g := gin.Default()
	gorestful.LoadFS(g)

	var v1 *gin.RouterGroup
	var am *gorestful.AuthMiddle
	auth := true //打卡认证
	if auth {
		type MyClaims struct {
			jwt.RegisteredClaims
			Username string
		}

		HeaderKey := "X-My-Token"

		am = &gorestful.AuthMiddle{
			URL:               "/login",
			HeaderKey:         HeaderKey,
			HeaderValuePrefix: "-",
			AuthMode: &gorestful.EmbedLogin{
				RouterGroup: g.Group("/"),
				CheckValid: func(e *gorestful.EmbedLogin, formMap map[string]string) (string, bool) {
					if formMap["user"] == "admin" && formMap["pass"] == "123456" {
						token := jwt.NewWithClaims(jwt.SigningMethodHS512, MyClaims{
							Username: formMap["user"],
						})

						if tokenString, err := token.SignedString(e.Key); err == nil {
							return tokenString, true
						} else {
							log.Println("jwt failed:", err)
						}

					}
					return "", false
				},
			},
		}

		v1 = g.Group("/api/v1", func(c *gin.Context) {
			if tokenString := c.Request.Header.Get(HeaderKey); tokenString != "" {
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
						return
					}
				}
			}

			c.AbortWithStatusJSON(403, map[string]interface{}{
				"code":    403,
				"message": "invalid auth",
			})
		})

		gorestful.AddAuthToGin(am)

	} else {
		v1 = g.Group("/api/v1")
	}

	res := &gorestful.Resource{
		Name: "user",
		//Fields: []gorestful.Field{
		//	{
		//		Name:      "ID",
		//		Type:      "uint",
		//		CloseEdit: true,
		//	},
		//	{
		//		Name: "email",
		//		Type: "string",
		//	},
		//	{
		//		Name:      "CreatedAt",
		//		Type:      "string",
		//		CloseEdit: true,
		//	},
		//},
		BlackFields:     []string{"CreatedAt"},
		ApiRouterGroup:  v1,
		PageRouterGroup: g.Group("/"),
		GetModel: func() interface{} {
			return &User{}
		},
		GetDb: func() *gorm.DB {
			return gdb
		},
		AuthMiddle: am,
	}

	if err = gorestful.AddResourceApiPageToGin(res); err != nil {
		panic(err)
	}

	g.Run(":9999")
}
