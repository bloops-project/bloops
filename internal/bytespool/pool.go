package bytespool

import (
	"bytes"
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func Get() (b *bytes.Buffer) {
	ifc := pool.Get()
	if ifc != nil {
		b = ifc.(*bytes.Buffer)
	}
	return
}

func Put(b *bytes.Buffer) {
	pool.Put(b)
}
