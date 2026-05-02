package main

import (
	"crypto/rand"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

func GeneratePadding() []byte {
	size := 100 + (randInt() % 924) // 100-1024 bytes
	padding := make([]byte, size)
	rand.Read(padding)
	return padding
}

func randInt() int {
	var b [1]byte
	rand.Read(b[:])
	return int(b[0])
}
