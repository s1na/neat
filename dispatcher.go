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
	c    net.Conn
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

func (req *Request) GetString(name string) (string, error) {
	t, err := req.GetArg(name)
	if err != nil {
		return "", err
	}
	retVal, ok := t.(string)
	if !ok {
		return "", fmt.Errorf("Arg %s is not castable to string.", name)
	}
	return retVal, nil
}

func (req *Request) GetInt(name string) (int, error) {
	t, err := req.GetArg(name)
	if err != nil {
		return 0, err
	}
	retVal, ok := t.(int)
	if !ok {
		return 0, fmt.Errorf("Arg %s is not castable to string.", name)
	}
	return retVal, nil
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
	sessions = make(map[string]interface{})
	return r
}

func (r *Router) Dispatch(msg []byte, c net.Conn) {
	num := int(msg[0])
	ctrl := r.table[num]

	var status uint8
	var resArgs []*Arg
	var res *Response

	if ctrl == nil {
		status = 255
		res = &Response{ctrl: nil, c: c, args: resArgs}
	} else {
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
		req := &Request{ctrl: ctrl, c: c, args: reqArgs}

		resArgs = make([]*Arg, len(ctrl.resArgs))
		for i, a := range ctrl.resArgs {
			resArgs[i] = &Arg{}
			*resArgs[i] = *a
		}
		res = &Response{ctrl: ctrl, c: c, args: resArgs}

		status = ctrl.f(req, res)
	}

	msg = make([]byte, 2, 100)
	msg[0] = byte(status)
	msg[1] = byte(len(resArgs))
	var argLen uint8 = 0
	for _, a := range res.args {
		if a.value != nil {
			switch a.type_ {
			case reflect.String:
				argLen = uint8(len(a.value.(string)))
				msg = append(msg, argLen)
				msg = append(msg, a.value.(string)...)
			case reflect.Int32:
				argLen = 4
				msg = append(msg, argLen)
				bs := bsInt32(a.value.(int32))
				msg = append(msg, bs...)
			}

		} else {
			msg = append(msg, 0)
		}
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
