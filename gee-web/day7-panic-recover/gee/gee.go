package gee

import (
	"html/template"
	"net/http"
	"path"
	"strings"
)

// HandlerFunc定义一个handler处理请求路由
// 参数修改为 *Context
type HandlerFunc func(c *Context)

// Group对象，还需要有访问Router的能力，为了方便，
// 我们可以在Group中，保存一个指针，指向Engine，
// 整个框架的所有资源都是由Engine统一协调的，那么就可以通过Engine间接地访问各种接口了。
type RouterGroup struct {
	prefix      string        // 用前缀区分分组
	middlewares []HandlerFunc // 用来处理该前缀对应的分组的方法集合
	parent      *RouterGroup  // 要支持分组嵌套 需要知道当前分组的父亲(parent)是谁
	engine      *Engine       // 方便访问router,所有的group共享一个Engine单例
}

// Engine implement the interface of ServeHTTP
// 里面有个静态变量router，不同的请求可以调用不同的处理逻辑
// 和路由有关的函数，都交给RouterGroup实现
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup // 保存全部的group

	htmlTemplates *template.Template // 将所有的模板加载进内存
	funcMap       template.FuncMap   // 自定义模板渲染函数
}

// gee.go
// Default use Logger() & Recovery middlewares
func Default() *Engine {
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}

// gee.Engine的构造函数
func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

// 创建新的路由分组
// 所有的group共享一个Engine单例
func (group *RouterGroup) Group(prefix string) *RouterGroup {
	engine := group.engine
	newGroup := &RouterGroup{
		prefix: group.prefix + prefix,
		parent: group,
		engine: engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup

}

// 给engine增加路由的handler
func (group *RouterGroup) addRoute(method string, prefix string, handler HandlerFunc) {
	engine := group.engine
	pattern := group.prefix + prefix // /v1 + /hello
	engine.router.addRoute(method, pattern, handler)
}

// GET defines the method to add GET request
// 调用GET可以给engin绑定一个请求为GET的路由，可以有多个这样的路由
// 实际上就是 "GET-/"或者"GET-hello"作为key
func (group *RouterGroup) GET(pattern string, handler HandlerFunc) {
	group.addRoute("GET", pattern, handler)
}

// POST defines the method to add POST request
func (group *RouterGroup) POST(pattern string, handler HandlerFunc) {
	group.addRoute("POST", pattern, handler)
}

// Run defines the method to start a http server
func (engine *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, engine)
}

// 修改了ServeHTTP的逻辑，将具体逻辑封装到handle函数，
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range engine.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	c := newContext(w, req)
	c.handlers = middlewares
	c.engine = engine // add
	engine.router.handle(c)
}

// 将给group增加传入的handler
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.middlewares = append(group.middlewares, middlewares...)
}

// 根据传入的相对路径（相对localhost的路径）先构造完整的绝对路径
// 如 group是/v1,相对路径是/assert，则拼接起来是 /vi/assert
// fs是映射的文件系统
func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(group.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		// Check if file exists and/or if we have permission to access it
		if _, err := fs.Open(file); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// 解析请求的地址，映射到服务器上文件的真实地址，交给http.FileServer处理就好了。
// 将磁盘上的某个文件夹root映射到路由relativePath。
// eg:
// r.Static("/assets", "/usr/geektutu/blog/static")
// 或相对路径 r.Static("/assets", "./static")
func (group *RouterGroup) Static(relativePath string, root string) {
	// 将相对路径和文件系统交给createStaticHandler，构造一个handler
	handler := group.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	// Register GET handlers
	group.GET(urlPattern, handler)

}

func (engine *Engine) SetFuncMap(funcMap template.FuncMap) {
	engine.funcMap = funcMap
}

func (engine *Engine) LoadHTMLGlob(pattern string) {
	engine.htmlTemplates = template.Must(template.New("").Funcs(engine.funcMap).ParseGlob(pattern))
}
