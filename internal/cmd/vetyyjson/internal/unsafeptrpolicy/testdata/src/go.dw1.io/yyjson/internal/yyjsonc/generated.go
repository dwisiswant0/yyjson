package yyjsonc

import "unsafe"

func generatedPointer(addr uintptr) unsafe.Pointer {
	return unsafe.Pointer(addr)
}
