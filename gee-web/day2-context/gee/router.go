package gee

import (
	"log"
	"net/http"
)

/*
	将和路由相关的方法和结构提取了出来，
	方便我们下一次对 router 的功能进行增强，
	例如提供动态路由的支持。
	router 的 handle 方法作了一个细微的调整，即 handler 的参数，变成了 Context。
*/
type router struct {
	handlers map[string]HandlerFunc
}

// router的构造函数
func newRouter() *router {
	return &router{
		handlers: make(map[string]HandlerFunc),
	}
}

// 增加路由规则
func (r *router)addRouter(method string,pattern string,handler HandlerFunc){
	log.Printf("Route %4s - %4s",method,pattern)
	key := method + "-" +pattern
	r.handlers[key] = handler
}

// handle函数执行路由的跳转，不同的key对应不同的路由规则，这里的handler是一个函数
func (r *router)handle(c *Context){
	key := c.Method + "-" + c.Path
	if handler,ok := r.handlers[key];ok{
		handler(c)
	}else{
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}