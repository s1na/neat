package neat

import (
	"net"
)

var (
	sessions map[string]interface{}
)

func (req *Request) GetSession() interface{} {

	return sessions[req.c.RemoteAddr().String()]
}

func InitSession(c net.Conn, s interface{}) {
	sessions[c.RemoteAddr().String()] = s
}
