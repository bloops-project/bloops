package strpool

import (
	"strings"
	"sync"
)

var strPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

func Get() (b *strings.Builder) {
	ifc := strPool.Get()
	if ifc != nil {
		b = ifc.(*strings.Builder)
	}
	return
}

func Put(b *strings.Builder) {
	strPool.Put(b)
}
