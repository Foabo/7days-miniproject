package gee

import (
	"net/http"
)

// HandlerFunc定义一个handler处理请求路由
// 参数修改为 *Context
type HandlerFunc func(c *Context)

// Engine implement the interface of ServeHTTP
// 里面有个静态变量router，不同的请求可以调用不同的处理逻辑
type Engine struct {
	router *router
}

// gee.Engine的构造函数
func New() *Engine {
	return &Engine{router: newRouter()}
}

// 给engine增加路由的handler
func (engine *Engine) addRouter(method string, pattern string, handler HandlerFunc) {
	engine.router.addRouter(method,pattern,handler)
}

// GET defines the method to add GET request
// 调用GET可以给engin绑定一个请求为GET的路由，可以有多个这样的路由
// 实际上就是 "GET-/"或者"GET-hello"作为key
func (engine *Engine) GET(pattern string, handler HandlerFunc) {
	engine.addRouter("GET", pattern, handler)
}

// POST defines the method to add POST request
func (engine *Engine) POST(pattern string, handler HandlerFunc) {
	engine.addRouter("POST", pattern, handler)
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}
// 修改了ServeHTTP的逻辑，将具体逻辑封装到handle函数，
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newContext(w,req)
	engine.router.handle(c)
}
