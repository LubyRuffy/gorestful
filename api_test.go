package gorestful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func getJson(url string) (map[string]interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var a map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&a)
	return a, err
}

func postJson(url string, data interface{}) (map[string]interface{}, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "text/json", &buf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var a map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&a)
	return a, err
}

func deleteJson(url string) (map[string]interface{}, error) {
	c := http.Client{}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var a map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&a)
	return a, err
}

type User struct {
	gorm.Model
	Email string `json:"email"`
}

func TestAddResourceToGin(t *testing.T) {
	cfg := &gorm.Config{}
	dbfile := filepath.Join(os.TempDir(), time.Now().Format("20060102150405.sqlite"))
	defer os.Remove(dbfile)
	//dsn := fmt.Sprintf("%s?journal_mode=%s&busy_timeout=%s", "a.sqlite", "WAL", "9999999")
	dsn := dbfile
	gdb, err := gorm.Open(sqlite.Open(dsn), cfg)
	assert.Nil(t, err)

	err = gdb.AutoMigrate(&User{})
	assert.Nil(t, err)

	g := gin.Default()
	res := &Resource{
		Name:           "user",
		ApiRouterGroup: g.Group("/api/v1"),
		GetDb: func(c *gin.Context) *gorm.DB {
			return gdb.Model(&User{})
		},
		GetModel: func() interface{} {
			return &User{}
		},
	}
	AddResourceApiToGin(res)

	// 启动服务
	s := httptest.NewServer(g)
	defer s.Close()

	// 读列表
	r, err := getJson(s.URL + res.ApiUrl())
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, float64(0), r["data"].(map[string]interface{})["count"])

	// 新增
	r, err = postJson(s.URL+res.ApiUrl(), &User{
		Email: "a@a.com",
	})
	assert.Nil(t, err)
	assert.NotNil(t, r)
	cid, ok := r["data"]
	assert.True(t, ok)
	assert.Greater(t, cid, float64(0))
	// 读列表
	r, err = getJson(s.URL + res.ApiUrl())
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, float64(1), r["data"].(map[string]interface{})["count"])
	assert.Equal(t, "a@a.com", r["data"].(map[string]interface{})["list"].([]interface{})[0].(map[string]interface{})["email"])
	id := r["data"].(map[string]interface{})["list"].([]interface{})[0].(map[string]interface{})["ID"]

	// 查看
	r, err = getJson(s.URL + fmt.Sprintf("%s/%v", res.ApiUrl(), id))
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, "a@a.com", r["data"].(map[string]interface{})["email"])

	// 修改
	r, err = postJson(s.URL+fmt.Sprintf("%s/%v", res.ApiUrl(), id), &User{
		Email: "b@a.com",
	})
	assert.Nil(t, err)
	assert.NotNil(t, r)
	r, err = getJson(s.URL + fmt.Sprintf("%s/%v", res.ApiUrl(), id))
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, "b@a.com", r["data"].(map[string]interface{})["email"])

	// 删除
	r, err = deleteJson(s.URL + fmt.Sprintf("%s/%v", res.ApiUrl(), id))
	assert.Nil(t, err)
	assert.NotNil(t, r)
	// 读列表
	r, err = getJson(s.URL + res.ApiUrl())
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, float64(0), r["data"].(map[string]interface{})["count"])
}
