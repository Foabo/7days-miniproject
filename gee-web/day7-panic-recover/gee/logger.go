package gee

import (
	"log"
	"time"
)
// 记录请求到响应所花费的时间
func Logger()HandlerFunc{
	return func(c *Context) {
		// 开始时间
		startTime := time.Now()

		c.Next()

		// 结束时间
		log.Printf("[%d] %s in %v", c.StatusCode, c.Req.RequestURI, time.Since(startTime))

	}
}
