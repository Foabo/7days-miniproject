package geerpc

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type methodType struct {
	method    reflect.Method // 方法本身
	ArgType   reflect.Type   // 第一个参数类型，是客户端发给服务端的参数
	ReplyType reflect.Type   // 第二个参数类型，服务端返回响应的那个参数
	numCalls  uint64         // 统计方法调用次数
}

func (m *methodType) NumCalls() uint64 {
	return atomic.LoadUint64(&m.numCalls)
}

// reflect.Elem()函数用于获取接口v包含的值或指针v指向的值
// reflect.Type 是接口
// reflect.Value 是结构体
func (m *methodType) newArgv() reflect.Value {
	var argv reflect.Value
	// arg 可能是个指针类型，也可能是个值类型
	// 指针类型和值类型创建实例的方式有细微区别。
	// Kind()检查类型
	if m.ArgType.Kind() == reflect.Ptr {
		// reflect.New传一个reflect.Type，返回一个reflect.Value
		argv = reflect.New(m.ArgType.Elem())
	} else {
		// 如果是值类型
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

func (m *methodType) newReplyv() reflect.Value {
	// reply一定得是指针类型
	replyv := reflect.New(m.ReplyType.Elem())
	switch m.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type service struct {
	name   string                 // 映射的结构体名称
	typ    reflect.Type           // 结构体的类型
	rcvr   reflect.Value          // 结构体实例本身，保留 rcvr 是因为在调用时需要 rcvr 作为第 0 个参数
	method map[string]*methodType // 存储映射的结构体的所有符合条件的方法。
}

// NewService 返回新的service实例的指针
// 传入的一般是某个结构体
func newService(recvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(recvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	// fmt.Println(s.name) s.name是结构体的名称 如 Foo
	s.typ = reflect.TypeOf(recvr)
	// fmt.Println(s.typ) s.typ是结构体的类型，如*geerpc.Foo
	// 如果方法是导出的
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.name)
	}
	s.registerMethods()
	return s
}

// 为一次service注册rpc调用的方法
// 首先是要过滤掉不是rpc调用的方法
// 最后给service.method添加methodType结构体
func (s *service) registerMethods() {
	s.method = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		method := s.typ.Method(i) // reflect.Type接口的Method方法返回type的第i个方法
		mType := method.Type      // 方法类型，即函数签名
		// fmt.Println(mType) 如func(*geerpc.Foo, geerpc.Args, *int) error
		// 传入参数不等于3或者返回值不等于1
		// 也即日餐包括自身和两个参数argv和reply，返回值是error
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			// 说明这个方法不是rpc的方法
			continue
		}
		// 如果返回值不是error接口
		// (*error)(nil) 写法是获取error接口类型
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		// 如果这个方法的参数和返回值不是导出的或者不是内置的，这个方法也是没用的
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.method[method.Name] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.name, method.Name)

	}
}

// service 调用一次这个方法
func (s *service) call(m *methodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	// f是要调用的参数
	f := m.method.Func
	// f.Call()第一个参数是结构体本身， 看成是this指针，后面的参数是方法的参数
	// 会把具体的response赋值给replyv指针
	// 返回值是一个数组
	returnValues := f.Call([]reflect.Value{s.rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}

// 判断是否是导出类型(public)或者是否是内置类型
func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}
