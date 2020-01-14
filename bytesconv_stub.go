// +build purego

package httprouter

import (
	"unsafe"
)

// stringToBytes converts string to byte slice without a memory allocation.
func stringToBytes(s string) (b []byte) {
	sh := *(*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
	return b
}

// bytesToString converts byte slice to string without a memory allocation.
func bytesToString(s []byte) string {
	return *(*string)(unsafe.Pointer(&s))
}
