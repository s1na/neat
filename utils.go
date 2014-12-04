package neat

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"runtime"
)

func getFuncName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func bsInt32(val int32) []byte {
	tbs := make([]byte, 4)
	bs := make([]byte, 4)
	n := binary.PutVarint(tbs, int64(val))
	fmt.Println(tbs)
	for i := 0; i < n; i++ {
		bs[(4-n)+i] = tbs[i]
	}
	fmt.Println(byte(250))
	fmt.Println(bs)
	return bs
}
