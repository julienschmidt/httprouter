// +build !purego

package httprouter

func bytesToString(s []byte) string {
	return string(s)
}
