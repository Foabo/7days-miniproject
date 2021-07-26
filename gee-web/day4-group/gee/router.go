package gee

import (
	"log"
	"net/http"
	"strings"
)

/*
	将和路由相关的方法和结构提取了出来，
	方便我们下一次对 router 的功能进行增强，
	例如提供动态路由的支持。
	router 的 handle 方法作了一个细微的调整，即 handler 的参数，变成了 Context。
*/
type router struct {
	roots    map[string]*node       // 存储每种请求方式的Trie 树根节点
	handlers map[string]HandlerFunc // 存储每种请求方式的 HandlerFunc
}

// roots key eg, roots['GET'] roots['POST']
// handlers key eg, handlers['GET-/p/:lang/doc'], handlers['POST-/p/book']
// router的构造函数
func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),
		handlers: make(map[string]HandlerFunc),
	}
}

// 根据传入的待匹配的pattern，解析成string数组
func parsePattern(pattern string) []string {
	vs := strings.Split(pattern, "/")
	parts := make([]string, 0)
	// 判断一下vs是否符合要求
	for _, part := range vs {
		if part != "" {
			parts = append(parts, part)
			if part[0] == '*' { // 只允许传入一个*
				break
			}
		}
	}

	return parts
}

// 增加路由规则
func (r *router) addRoute(method string, pattern string, handler HandlerFunc) {

	parts := parsePattern(pattern)

	log.Printf("Route %4s - %4s", method, pattern)
	key := method + "-" + pattern

	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{} // 插入一个根节点
	}
	r.roots[method].insert(pattern, parts, 0)

	r.handlers[key] = handler
}

// 获取路由，参数为请求方法和请求路径
func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path) // 实际的路径解析的结果
	params := make(map[string]string) // 动态路径参数，只在当前查询内有效，有可能一次请求路径没有动态参数

	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}
	// 只有trie树上有这条路径，n才不为空,这个n有pattern的字段
	n := root.search(searchParts, 0)
	if n != nil {
		// trie树上的解析结果
		// n的pattern可能和path不一样
		parts := parsePattern(n.pattern)
		// 例如
		// /p/go/doc 匹配到 /p/:lang/doc，解析结果为：{lang: "go"}，
		// /static/css/geektutu.css 匹配到 /static/*filepath ，解析结果为 {filepath: "css/geektutu.css"}。
		for index, part := range parts {
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}

func (r *router) getRoutes(method string) []*node {
	root, ok := r.roots[method]
	if !ok {
		return nil
	}
	nodes := make([]*node, 0)
	root.travel(&nodes)
	return nodes
}

// handle函数执行路由的跳转，不同的key对应不同的路由规则，这里的handler是一个函数
func (r *router) handle(c *Context) {
	n, params := r.getRoute(c.Method, c.Path)
	if n != nil {
		c.Params = params
		key := c.Method + "-" + n.pattern
		log.Printf("key: %s \n", key)
		r.handlers[key](c) // handel方法
	} else {
		c.String(http.StatusNotFound, "404 NOT FOUND: %s\n", c.Path)
	}
}
