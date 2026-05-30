package other

import "unsafe"

type address uintptr

func badPointer(addr uintptr) unsafe.Pointer {
	return unsafe.Pointer(addr) // want "uintptr to unsafe.Pointer conversion"
}

func badNamedPointer(addr address) unsafe.Pointer {
	return unsafe.Pointer(addr) // want "uintptr to unsafe.Pointer conversion"
}

func notTheBindEscapeHatch(addr uintptr) unsafe.Pointer {
	return unsafe.Pointer(addr) // want "uintptr to unsafe.Pointer conversion"
}

func cPointer(addr uintptr) unsafe.Pointer {
	return unsafe.Pointer(addr) // want "uintptr to unsafe.Pointer conversion"
}
