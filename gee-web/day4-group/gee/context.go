package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

// 定义 Context
type Context struct {
	// origin objects
	Writer http.ResponseWriter
	Req    *http.Request
	// request info
	Path   string
	Method string
	// 提供对路由参数的访问。我们将解析后的参数存储到Params中，通过c.Param("lang")的方式获取到对应的值。
	Params map[string]string
	// response info
	StatusCode int
}

// Context 的构造函数
func newContext(w http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Writer: w,
		Req:    req,
		Path:   req.URL.Path,
		Method: req.Method,
	}
}

func (c *Context) Param(key string) string {
	value ,_ := c.Params[key]
	return value
}

// 表单数据
func (c *Context) PostForm(key string) string {
	// FormValue里面有 r.Form[key]，FormValue则返回这个key对应的value里面的第一个值
	return c.Req.FormValue(key)
}

// query是指请求的参数，一般是指URL中？后面的参数
func (c *Context) Query(key string) string {
	return c.Req.URL.Query().Get(key)
}

// 设置状态码
func (c *Context) Status(code int) {
	c.StatusCode = code
	// WriteHeader sends an HTTP response header with the provided
	// status code.
	c.Writer.WriteHeader(code)
}

// 设置头部的key value对
// 如c.SetHeader("Content-Type", "application/json")
func (c *Context) SetHeader(key string, value string) {
	c.Writer.Header().Set(key, value)
}

// 返回数据为text/plain类型的response
// 这里需要自定义的format 以便可以使用任意类型的value，类似于泛型的设计
func (c *Context) String(code int, format string, values ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	// 正确的调用顺序应该是Header().Set 然后WriteHeader() 最后是Write()
	c.Writer.Write([]byte(fmt.Sprintf(format, values...))) // Sprintf返回的是一个string
}

// 响应的数据为json格式
func (c *Context) JSON(code int, obj ...interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	//
	encoder := json.NewEncoder(c.Writer) // c.Writer是json数据的接收者
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), 500)
	}
}

// response写入数据,这些数据是字符数据
func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Writer.Write(data)
}

// 构造HTML响应
func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Writer.Write([]byte(html))
}
