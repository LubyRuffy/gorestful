package gorestful

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func (u User) GetID() uint {
	return u.ID
}

func TestAddResourceToGinRouter(t *testing.T) {
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
	prefix := "/api/v1"
	v1 := g.Group(prefix)
	AddResourceToGinRouter("user", v1, func() *gorm.DB {
		return gdb
	}, func() Model {
		return &User{}
	})

	// 启动服务
	s := httptest.NewServer(g)
	defer s.Close()

	// 读列表
	r, err := getJson(s.URL + prefix + "/user")
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, float64(0), r["data"].(map[string]interface{})["count"])

	// 新增
	r, err = postJson(s.URL+prefix+"/user", &User{
		Email: "a@a.com",
	})
	assert.Nil(t, err)
	assert.NotNil(t, r)
	// 读列表
	r, err = getJson(s.URL + prefix + "/user")
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, float64(1), r["data"].(map[string]interface{})["count"])

	// 删除
	r, err = deleteJson(s.URL + prefix + "/user/")
	assert.Nil(t, err)
	assert.NotNil(t, r)
	// 读列表
	r, err = getJson(s.URL + prefix + "/user")
	assert.Nil(t, err)
	assert.NotNil(t, r)
	assert.Equal(t, float64(1), r["data"].(map[string]interface{})["count"])
}