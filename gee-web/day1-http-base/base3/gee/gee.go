package gee

import (
	"fmt"
	"net/http"
)

// HandlerFunc定义一个handler处理请求路由
type HandlerFunc func(w http.ResponseWriter, req *http.Request)

// Engine implement the interface of ServeHTTP
// 里面有个静态变量router，不同的请求可以调用不同的处理逻辑
type Engine struct {
	router map[string]HandlerFunc
}

// gee.Engine的构造函数
func New() *Engine {
	return &Engine{router: make(map[string]HandlerFunc)}
}

// 给engine增加路由的handler
func (engine *Engine) addRouter(method string, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern
	engine.router[key] = handler
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

func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	key := req.Method + "-" + req.URL.Path
	if handler, ok := engine.router[key]; ok {
		handler(w, req)// 执行真正的处理逻辑，处理逻辑gee.go没有定义，在main.go才真正定义处理逻辑
	} else {
		fmt.Fprintf(w, "404 NOT FOUND: %s\n", req.URL)
	}
}
