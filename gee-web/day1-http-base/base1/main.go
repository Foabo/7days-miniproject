package main

import (
	"fmt"
	"log"
	"net/http"
)

/**
	设置了2个路由，/和/hello，分别绑定 indexHandler 和 helloHandler ， 根据不同的HTTP请求会调用不同的处理函数。
*/
func main(){
	http.HandleFunc("/",indexHandler)
	http.HandleFunc("/hello",helloHandler)
	// 启动http服务器,底层监听了tcp端口
	// 第二个参数是一个handler。
	// Handler是一个接口，需要实现方法 ServeHTTP ，
	// 也就是说，只要传入任何实现了 ServerHTTP 接口的实例，所有的HTTP请求，就都交给了该实例处理了
	log.Fatal(http.ListenAndServe(":9999",nil))
}
// handler echoes r.URL.Path
// curl -X GET localhost:9999
func indexHandler(w http.ResponseWriter, req *http.Request) {
	// Fprintf：来格式化并输出到 io.Writers 而不是 os.Stdout。
	fmt.Fprintf(w,"URL.Path = %q\n",req.URL.Path)
}
// handler echoes r.URL.Header
// curl -X GET localhost:9999/hello
func helloHandler(w http.ResponseWriter, req *http.Request) {
	for k,v := range req.Header{
		fmt.Fprintf(w,"Header[%q] = %q\n",k,v)
	}
}


