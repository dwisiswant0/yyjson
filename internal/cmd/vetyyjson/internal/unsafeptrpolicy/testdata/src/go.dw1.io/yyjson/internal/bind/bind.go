package bind

import "unsafe"

func cPointer(addr uintptr) unsafe.Pointer {
	return unsafe.Pointer(addr)
}

func badBindPointer(addr uintptr) unsafe.Pointer {
	return unsafe.Pointer(addr) // want "uintptr to unsafe.Pointer conversion"
}

func pointerToUnsafePointer(p *byte) unsafe.Pointer {
	return unsafe.Pointer(p)
}
