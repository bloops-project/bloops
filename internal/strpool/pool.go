package strpool

import (
	"strings"
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

func Get() (b *strings.Builder) {
	ifc := pool.Get()
	if ifc != nil {
		b = ifc.(*strings.Builder)
	}
	return
}

func Put(b *strings.Builder) {
	pool.Put(b)
}
