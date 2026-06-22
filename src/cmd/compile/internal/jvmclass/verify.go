package jvmclass

// Verify performs a lightweight classfile marker check.
func Verify(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 0xca &&
		data[1] == 0xfe &&
		data[2] == 0xba &&
		data[3] == 0xbe
}
