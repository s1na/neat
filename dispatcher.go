package neat

import (
	//"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
)

type CtrlFunc func(req *Request, res *Response) uint8

type CtrlMetaData struct {
	num     int
	f       CtrlFunc
	reqArgs []*Arg
	resArgs []*Arg
}

type Request struct {
	ctrl *CtrlMetaData
	args []*Arg
}

type Response struct {
	ctrl *CtrlMetaData
	c    net.Conn
	args []*Arg
}

type Arg struct {
	name            string
	type_           reflect.Kind
	required        bool
	default_, value interface{}
}

type Router struct {
	table map[int]*CtrlMetaData
}

func NewArg(name string, type_ reflect.Kind) *Arg {
	return &Arg{name: name, type_: type_}
}

func (req *Request) GetArg(name string) (interface{}, error) {
	for _, a := range req.args {
		if name == a.name {
			if t := reflect.ValueOf(a.value).Kind(); t != a.type_ {
				return nil, fmt.Errorf(
					"dispatcher: request argument %s(%s) is not of valid type(%s) for controller %s.",
					name, t, a.type_, getFuncName(req.ctrl.f),
				)
			}
			return a.value, nil
		}
	}
	return nil, fmt.Errorf(
		"dispatcher: request argument %s is not defined for controller %s.",
		name, getFuncName(req.ctrl.f),
	)
}

func (res *Response) SetArg(name string, value interface{}) error {
	for _, a := range res.args {
		if name == a.name {
			if t := reflect.ValueOf(value).Kind(); t != a.type_ {
				return fmt.Errorf(
					"dispatcher: response argument %s(%s) is not of valid type(%s) for controller %s.",
					name, t, a.type_, getFuncName(res.ctrl.f),
				)
			}
			a.value = value
			return nil
		}
	}
	return fmt.Errorf(
		"dispatcher: response argument %s is not defined for controller %s.",
		name, getFuncName(res.ctrl.f),
	)
}

func NewRouter() *Router {
	r := &Router{table: make(map[int]*CtrlMetaData)}
	return r
}

func (r *Router) Dispatch(msg []byte, c net.Conn) {
	num := int(msg[0])
	ctrl := r.table[num]
	reqArgs := make([]*Arg, len(ctrl.reqArgs))
	for i, a := range ctrl.reqArgs {
		reqArgs[i] = &Arg{}
		*reqArgs[i] = *a
	}
	i := 1
	l := 0
	for _, a := range reqArgs {
		l = int(msg[i])
		i++
		switch a.type_ {
		case reflect.String:
			a.value = string(msg[i : i+l])
		case reflect.Int32:
			val, _ := binary.Varint(msg[i : i+l])
			a.value = int32(val)
		}
		i += l
	}
	req := &Request{ctrl: ctrl, args: reqArgs}

	resArgs := make([]*Arg, len(ctrl.resArgs))
	for i, a := range ctrl.resArgs {
		resArgs[i] = &Arg{}
		*resArgs[i] = *a
	}
	res := &Response{ctrl: ctrl, c: c, args: resArgs}
	status := ctrl.f(req, res)

	msg = make([]byte, 1, 100)
	msg[0] = byte(status)
	var argLen uint8 = 0
	var mi uint8 = 2
	for _, a := range res.args {
		switch a.type_ {
		case reflect.String:
			argLen = uint8(len(a.value.(string)))
			msg = append(msg, argLen)
			mi++
			msg = append(msg, a.value.(string)...)
		case reflect.Int32:
			argLen = 4
		}

		mi += argLen
	}

	c.Write(msg)
}

func (r *Router) Register(num int, f CtrlFunc) *CtrlMetaData {
	ctrl := &CtrlMetaData{num: num, f: f}
	r.table[num] = ctrl
	return ctrl
}

func (ctrl *CtrlMetaData) SetReqArgs(args ...*Arg) *CtrlMetaData {
	ctrl.reqArgs = args
	return ctrl
}

func (ctrl *CtrlMetaData) SetResArgs(args ...*Arg) *CtrlMetaData {
	ctrl.resArgs = args
	return ctrl
}
