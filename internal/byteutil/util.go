package byteutil

import (
	"encoding/binary"
	"unsafe"
)

func EncodeInt64ToBytes(id int64) []byte {
	b := make([]byte, 64)
	binary.BigEndian.PutUint64(b, uint64(id))
	return b
}

func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
