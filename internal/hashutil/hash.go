package hashutil

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/bloops-games/bloops/internal/bytespool"
	"strconv"
	"time"
)

func SerializedSha1FromTime() string {
	buf := bytespool.Get()
	defer func() {
		buf.Reset()
		bytespool.Put(buf)
	}()
	buf.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
	hash := sha1.New()
	hash.Write(buf.Bytes())
	return hex.EncodeToString(hash.Sum(nil))
}

func Sha1FromTimestamp() ([20]byte, error) {
	var hash [20]byte
	b, err := time.Now().MarshalBinary()
	if err != nil {
		return hash, fmt.Errorf("marshal binary time.now(): %v", err)
	}

	hash = sha1.Sum(b)
	return hash, nil
}
