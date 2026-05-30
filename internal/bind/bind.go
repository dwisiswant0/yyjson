package bind

import (
	"fmt"
	"math"
	"strings"
	"unsafe"

	c "go.dw1.io/yyjson/internal/yyjsonc"
	"modernc.org/libc"
)

const ReadInsitu = 1 << 0

// Must match YYJSON_PADDING_SIZE in src/yyjson.h.
const readPaddingSize = 4

const incrBufferExtra = 64

type Doc uintptr
type Val uintptr
type MutDoc uintptr
type MutVal uintptr
type IncrState uintptr

type ReadError struct {
	Code     uint32
	Message  string
	Position uint64
}

func (e *ReadError) Error() string {
	if e == nil {
		return ""
	}
	if e.Position != 0 {
		return fmt.Sprintf("yyjson read error: code=%d pos=%d: %s", e.Code, e.Position, e.Message)
	}
	return fmt.Sprintf("yyjson read error: code=%d: %s", e.Code, e.Message)
}

type WriteError struct {
	Code    uint32
	Message string
}

func (e *WriteError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("yyjson write error: code=%d: %s", e.Code, e.Message)
}

type PointerError struct {
	Code     uint32
	Message  string
	Position uint64
}

func (e *PointerError) Error() string {
	if e == nil {
		return ""
	}
	if e.Position != 0 {
		return fmt.Sprintf("yyjson pointer error: code=%d pos=%d: %s", e.Code, e.Position, e.Message)
	}
	return fmt.Sprintf("yyjson pointer error: code=%d: %s", e.Code, e.Message)
}

type PatchError struct {
	Code    uint32
	Index   uint64
	Message string
	Pointer PointerError
}

func (e *PatchError) Error() string {
	if e == nil {
		return ""
	}
	if e.Pointer.Code != 0 {
		return fmt.Sprintf("yyjson patch error: code=%d op=%d: %s: %s", e.Code, e.Index, e.Message, e.Pointer.Error())
	}
	return fmt.Sprintf("yyjson patch error: code=%d op=%d: %s", e.Code, e.Index, e.Message)
}

type ReadResult struct {
	Doc Doc
	Buf uintptr
}

func Version() uint32 {
	tls := libc.NewTLS()
	defer tls.Close()
	return uint32(c.Xyyjson_version(tls))
}

func Read(data []byte, flags uint32) (ReadResult, *ReadError) {
	tls := libc.NewTLS()
	defer tls.Close()

	ptr, err := allocBytes(tls, data)
	if err != nil {
		return ReadResult{}, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	keepBuf := flags&ReadInsitu != 0
	if !keepBuf {
		defer libc.Xfree(tls, ptr)
	}

	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_read_err{})))
	if errp == 0 {
		if keepBuf {
			libc.Xfree(tls, ptr)
		}
		return ReadResult{}, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_MEMORY_ALLOCATION), Message: "allocate read error"}
	}
	defer libc.Xfree(tls, errp)

	doc := c.Xyyjson_read_opts(tls, ptr, c.Tusize(len(data)), c.Tyyjson_read_flag(flags), 0, errp)
	if doc == 0 {
		if keepBuf {
			libc.Xfree(tls, ptr)
		}
		return ReadResult{}, readError(readErrorAt(errp))
	}
	res := ReadResult{Doc: Doc(doc)}
	if keepBuf {
		res.Buf = ptr
	}
	return res, nil
}

func ReadFile(path string, flags uint32) (Doc, *ReadError) {
	if hasNULByte(path) {
		return 0, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_INVALID_PARAMETER), Message: "input path contains NUL byte"}
	}

	tls := libc.NewTLS()
	defer tls.Close()

	cpath, err := libc.CString(path)
	if err != nil {
		return 0, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	defer libc.Xfree(tls, cpath)

	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_read_err{})))
	if errp == 0 {
		return 0, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_MEMORY_ALLOCATION), Message: "allocate read error"}
	}
	defer libc.Xfree(tls, errp)

	doc := c.Xyyjson_read_file(tls, cpath, c.Tyyjson_read_flag(flags), 0, errp)
	if doc == 0 {
		return 0, readError(readErrorAt(errp))
	}
	return Doc(doc), nil
}

func NewIncrState(maxLen int, flags uint32) (IncrState, uintptr, *ReadError) {
	if maxLen <= 0 {
		return 0, 0, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_INVALID_PARAMETER), Message: "incremental buffer length must be positive"}
	}
	if maxLen > math.MaxInt-incrBufferExtra {
		return 0, 0, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_INVALID_PARAMETER), Message: "incremental buffer length is too large"}
	}
	size := maxLen + incrBufferExtra
	tls := libc.NewTLS()
	defer tls.Close()
	buf := libc.Xmalloc(tls, c.Tsize_t(size))
	if buf == 0 {
		return 0, 0, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_MEMORY_ALLOCATION), Message: fmt.Sprintf("allocate %d bytes", size)}
	}
	clear(cBytes(buf, size))
	flags |= ReadInsitu
	state := c.Xyyjson_incr_new(tls, buf, c.Tsize_t(maxLen), c.Tyyjson_read_flag(flags), 0)
	if state == 0 {
		libc.Xfree(tls, buf)
		return 0, 0, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_MEMORY_ALLOCATION), Message: "allocate incremental reader state"}
	}
	return IncrState(state), buf, nil
}

func FreeIncrState(state IncrState, buf uintptr) {
	tls := libc.NewTLS()
	defer tls.Close()
	if state != 0 {
		c.Xyyjson_incr_free(tls, uintptr(state))
	}
	if buf != 0 {
		libc.Xfree(tls, buf)
	}
}

func IncrAppend(buf uintptr, off int, data []byte) {
	if len(data) == 0 {
		return
	}
	copy(cBytesAt(buf, off, len(data)), data)
}

func IncrRead(state IncrState, available int) (Doc, *ReadError) {
	tls := libc.NewTLS()
	defer tls.Close()
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_read_err{})))
	if errp == 0 {
		return 0, &ReadError{Code: uint32(c.YYJSON_READ_ERROR_MEMORY_ALLOCATION), Message: "allocate read error"}
	}
	defer libc.Xfree(tls, errp)
	doc := c.Xyyjson_incr_read(tls, uintptr(state), c.Tsize_t(available), errp)
	if doc == 0 {
		return 0, readError(readErrorAt(errp))
	}
	return Doc(doc), nil
}

func FreeDoc(doc Doc, buf uintptr) {
	tls := libc.NewTLS()
	defer tls.Close()
	if doc != 0 {
		c.Xyyjson_go_doc_free(tls, uintptr(doc))
	}
	if buf != 0 {
		libc.Xfree(tls, buf)
	}
}

func DocRoot(doc Doc) Val {
	return Val(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_doc_get_root(tls, uintptr(doc))
	}))
}

func DocReadSize(doc Doc) uint64 {
	return uint64(withTLSSize(func(tls *libc.TLS) c.Tsize_t {
		return c.Xyyjson_go_doc_get_read_size(tls, uintptr(doc))
	}))
}

func DocValCount(doc Doc) uint64 {
	return uint64(withTLSSize(func(tls *libc.TLS) c.Tsize_t {
		return c.Xyyjson_go_doc_get_val_count(tls, uintptr(doc))
	}))
}

func WriteDoc(doc Doc, flags uint32) ([]byte, *WriteError) {
	return write(func(tls *libc.TLS, lenp, errp uintptr) uintptr {
		return c.Xyyjson_write_opts(tls, uintptr(doc), c.Tyyjson_write_flag(flags), 0, lenp, errp)
	})
}

func WriteVal(val Val, flags uint32) ([]byte, *WriteError) {
	return write(func(tls *libc.TLS, lenp, errp uintptr) uintptr {
		return c.Xyyjson_val_write_opts(tls, uintptr(val), c.Tyyjson_write_flag(flags), 0, lenp, errp)
	})
}

func WriteDocFile(path string, doc Doc, flags uint32) *WriteError {
	return writeFile(path, func(tls *libc.TLS, pathp, errp uintptr) uint8 {
		return c.Xyyjson_write_file(tls, pathp, uintptr(doc), c.Tyyjson_write_flag(flags), 0, errp)
	})
}

func ValType(val Val) uint8 {
	return withTLSUint8(func(tls *libc.TLS) uint8 { return c.Xyyjson_go_get_type(tls, uintptr(val)) })
}

func ValSubtype(val Val) uint8 {
	return withTLSUint8(func(tls *libc.TLS) uint8 { return c.Xyyjson_go_get_subtype(tls, uintptr(val)) })
}

func ValTypeDesc(val Val) string {
	return withTLSString(func(tls *libc.TLS) uintptr { return c.Xyyjson_go_get_type_desc(tls, uintptr(val)) })
}

func ValRaw(val Val) string {
	return copyCStringLen(
		withTLSUintptr(func(tls *libc.TLS) uintptr { return c.Xyyjson_go_get_raw(tls, uintptr(val)) }),
		int(ValLen(val)),
	)
}

func ValBool(val Val) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 { return c.Xyyjson_go_get_bool(tls, uintptr(val)) }) != 0
}

func ValUint(val Val) uint64 {
	return uint64(withTLSUint64(func(tls *libc.TLS) c.Tuint64_t { return c.Xyyjson_go_get_uint(tls, uintptr(val)) }))
}

func ValInt(val Val) int64 {
	return int64(withTLSInt64(func(tls *libc.TLS) c.Tint64_t { return c.Xyyjson_go_get_sint(tls, uintptr(val)) }))
}

func ValFloat(val Val) float64 {
	return withTLSFloat64(func(tls *libc.TLS) float64 { return c.Xyyjson_go_get_real(tls, uintptr(val)) })
}

func ValNum(val Val) float64 {
	return withTLSFloat64(func(tls *libc.TLS) float64 { return c.Xyyjson_go_get_num(tls, uintptr(val)) })
}

func ValString(val Val) string {
	return copyCStringLen(
		withTLSUintptr(func(tls *libc.TLS) uintptr { return c.Xyyjson_go_get_str(tls, uintptr(val)) }),
		int(ValLen(val)),
	)
}

func ValLen(val Val) uint64 {
	return uint64(withTLSSize(func(tls *libc.TLS) c.Tsize_t { return c.Xyyjson_go_get_len(tls, uintptr(val)) }))
}

func ValEqualsString(val Val, s string) bool {
	tls := libc.NewTLS()
	defer tls.Close()
	ptr, err := allocBytes(tls, []byte(s))
	if err != nil {
		return false
	}
	defer libc.Xfree(tls, ptr)
	return c.Xyyjson_go_equals_strn(tls, uintptr(val), ptr, c.Tsize_t(len(s))) != 0
}

func ValEqual(lhs, rhs Val) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_equals(tls, uintptr(lhs), uintptr(rhs))
	}) != 0
}

func ArrLen(arr Val) uint64 {
	return uint64(withTLSSize(func(tls *libc.TLS) c.Tsize_t { return c.Xyyjson_go_arr_size(tls, uintptr(arr)) }))
}

func ArrGet(arr Val, idx uint64) Val {
	return Val(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_arr_get(tls, uintptr(arr), c.Tsize_t(idx))
	}))
}

func ArrValues(arr Val) []Val {
	tls := libc.NewTLS()
	defer tls.Close()
	var iter c.Tyyjson_arr_iter
	if c.Xyyjson_go_arr_iter_init(tls, uintptr(arr), uintptr(unsafe.Pointer(&iter))) == 0 {
		return nil
	}
	values := make([]Val, 0, int(iter.Fmax))
	for {
		val := c.Xyyjson_go_arr_iter_next(tls, uintptr(unsafe.Pointer(&iter)))
		if val == 0 {
			return values
		}
		values = append(values, Val(val))
	}
}

type Member struct {
	Key Val
	Val Val
}

func ObjLen(obj Val) uint64 {
	return uint64(withTLSSize(func(tls *libc.TLS) c.Tsize_t { return c.Xyyjson_go_obj_size(tls, uintptr(obj)) }))
}

func ObjGet(obj Val, key string) Val {
	tls := libc.NewTLS()
	defer tls.Close()
	ptr, err := allocBytes(tls, []byte(key))
	if err != nil {
		return 0
	}
	defer libc.Xfree(tls, ptr)
	return Val(c.Xyyjson_go_obj_getn(tls, uintptr(obj), ptr, c.Tsize_t(len(key))))
}

func ObjMembers(obj Val) []Member {
	tls := libc.NewTLS()
	defer tls.Close()
	var iter c.Tyyjson_obj_iter
	if c.Xyyjson_go_obj_iter_init(tls, uintptr(obj), uintptr(unsafe.Pointer(&iter))) == 0 {
		return nil
	}
	members := make([]Member, 0, int(iter.Fmax))
	for {
		key := c.Xyyjson_go_obj_iter_next(tls, uintptr(unsafe.Pointer(&iter)))
		if key == 0 {
			return members
		}
		val := c.Xyyjson_go_obj_iter_get_val(tls, key)
		members = append(members, Member{Key: Val(key), Val: Val(val)})
	}
}

func Pointer(val Val, ptr string) (Val, *PointerError) {
	tls := libc.NewTLS()
	defer tls.Close()
	cptr, err := allocBytes(tls, []byte(ptr))
	if err != nil {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	defer libc.Xfree(tls, cptr)
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_ptr_err{})))
	if errp == 0 {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: "allocate pointer error"}
	}
	defer libc.Xfree(tls, errp)
	out := c.Xyyjson_go_ptr_getx(tls, uintptr(val), cptr, c.Tsize_t(len(ptr)), errp)
	if out == 0 {
		return 0, pointerError(pointerErrorAt(errp))
	}
	return Val(out), nil
}

func DocPointer(doc Doc, ptr string) (Val, *PointerError) {
	tls := libc.NewTLS()
	defer tls.Close()
	cptr, err := allocBytes(tls, []byte(ptr))
	if err != nil {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	defer libc.Xfree(tls, cptr)
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_ptr_err{})))
	if errp == 0 {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: "allocate pointer error"}
	}
	defer libc.Xfree(tls, errp)
	out := c.Xyyjson_go_doc_ptr_getx(tls, uintptr(doc), cptr, c.Tsize_t(len(ptr)), errp)
	if out == 0 {
		return 0, pointerError(pointerErrorAt(errp))
	}
	return Val(out), nil
}

func NewMutDoc() MutDoc {
	return MutDoc(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_mut_doc_new(tls, 0)
	}))
}

func FreeMutDoc(doc MutDoc) {
	tls := libc.NewTLS()
	defer tls.Close()
	c.Xyyjson_mut_doc_free(tls, uintptr(doc))
}

func MutDocRoot(doc MutDoc) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_mut_doc_get_root(tls, uintptr(doc))
	}))
}

func MutDocSetRoot(doc MutDoc, root MutVal) {
	tls := libc.NewTLS()
	defer tls.Close()
	c.Xyyjson_go_mut_doc_set_root(tls, uintptr(doc), uintptr(root))
}

func MutDocFromDoc(doc Doc) MutDoc {
	return MutDoc(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_doc_mut_copy(tls, uintptr(doc), 0)
	}))
}

func MutDocToDoc(doc MutDoc) Doc {
	return Doc(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_mut_doc_imut_copy(tls, uintptr(doc), 0)
	}))
}

func MutValToDoc(val MutVal) Doc {
	return Doc(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_mut_val_imut_copy(tls, uintptr(val), 0)
	}))
}

func WriteMutDoc(doc MutDoc, flags uint32) ([]byte, *WriteError) {
	return write(func(tls *libc.TLS, lenp, errp uintptr) uintptr {
		return c.Xyyjson_mut_write_opts(tls, uintptr(doc), c.Tyyjson_write_flag(flags), 0, lenp, errp)
	})
}

func WriteMutVal(val MutVal, flags uint32) ([]byte, *WriteError) {
	return write(func(tls *libc.TLS, lenp, errp uintptr) uintptr {
		return c.Xyyjson_mut_val_write_opts(tls, uintptr(val), c.Tyyjson_write_flag(flags), 0, lenp, errp)
	})
}

func WriteMutDocFile(path string, doc MutDoc, flags uint32) *WriteError {
	return writeFile(path, func(tls *libc.TLS, pathp, errp uintptr) uint8 {
		return c.Xyyjson_mut_write_file(tls, pathp, uintptr(doc), c.Tyyjson_write_flag(flags), 0, errp)
	})
}

func MutNull(doc MutDoc) MutVal  { return mutNew(doc, c.Xyyjson_go_mut_null) }
func MutTrue(doc MutDoc) MutVal  { return mutNew(doc, c.Xyyjson_go_mut_true) }
func MutFalse(doc MutDoc) MutVal { return mutNew(doc, c.Xyyjson_go_mut_false) }
func MutArray(doc MutDoc) MutVal { return mutNew(doc, c.Xyyjson_go_mut_arr) }
func MutObject(doc MutDoc) MutVal {
	return mutNew(doc, c.Xyyjson_go_mut_obj)
}

func MutBool(doc MutDoc, v bool) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_mut_bool(tls, uintptr(doc), boolByte(v))
	}))
}

func MutUint(doc MutDoc, v uint64) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_mut_uint(tls, uintptr(doc), c.Tuint64_t(v))
	}))
}

func MutInt(doc MutDoc, v int64) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_mut_sint(tls, uintptr(doc), c.Tint64_t(v))
	}))
}

func MutReal(doc MutDoc, v float64) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_mut_real(tls, uintptr(doc), v)
	}))
}

func MutString(doc MutDoc, s string) MutVal {
	tls := libc.NewTLS()
	defer tls.Close()
	ptr, err := allocBytes(tls, []byte(s))
	if err != nil {
		return 0
	}
	defer libc.Xfree(tls, ptr)
	return MutVal(c.Xyyjson_go_mut_strncpy(tls, uintptr(doc), ptr, c.Tsize_t(len(s))))
}

func MutRaw(doc MutDoc, s string) MutVal {
	tls := libc.NewTLS()
	defer tls.Close()
	ptr, err := allocBytes(tls, []byte(s))
	if err != nil {
		return 0
	}
	defer libc.Xfree(tls, ptr)
	return MutVal(c.Xyyjson_go_mut_rawncpy(tls, uintptr(doc), ptr, c.Tsize_t(len(s))))
}

func MutValType(val MutVal) uint8 {
	return withTLSUint8(func(tls *libc.TLS) uint8 { return c.Xyyjson_go_mut_get_type(tls, uintptr(val)) })
}

func MutValSubtype(val MutVal) uint8 {
	return withTLSUint8(func(tls *libc.TLS) uint8 { return c.Xyyjson_go_mut_get_subtype(tls, uintptr(val)) })
}

func MutValTypeDesc(val MutVal) string {
	return withTLSString(func(tls *libc.TLS) uintptr { return c.Xyyjson_go_mut_get_type_desc(tls, uintptr(val)) })
}

func MutValString(val MutVal) string {
	return copyCStringLen(
		withTLSUintptr(func(tls *libc.TLS) uintptr { return c.Xyyjson_go_mut_get_str(tls, uintptr(val)) }),
		int(MutValLen(val)),
	)
}

func MutValRaw(val MutVal) string {
	return copyCStringLen(
		withTLSUintptr(func(tls *libc.TLS) uintptr { return c.Xyyjson_go_mut_get_raw(tls, uintptr(val)) }),
		int(MutValLen(val)),
	)
}

func MutValBool(val MutVal) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 { return c.Xyyjson_go_mut_get_bool(tls, uintptr(val)) }) != 0
}

func MutValUint(val MutVal) uint64 {
	return uint64(withTLSUint64(func(tls *libc.TLS) c.Tuint64_t { return c.Xyyjson_go_mut_get_uint(tls, uintptr(val)) }))
}

func MutValInt(val MutVal) int64 {
	return int64(withTLSInt64(func(tls *libc.TLS) c.Tint64_t { return c.Xyyjson_go_mut_get_sint(tls, uintptr(val)) }))
}

func MutValFloat(val MutVal) float64 {
	return withTLSFloat64(func(tls *libc.TLS) float64 { return c.Xyyjson_go_mut_get_real(tls, uintptr(val)) })
}

func MutValNum(val MutVal) float64 {
	return withTLSFloat64(func(tls *libc.TLS) float64 { return c.Xyyjson_go_mut_get_num(tls, uintptr(val)) })
}

func MutValLen(val MutVal) uint64 {
	return uint64(withTLSSize(func(tls *libc.TLS) c.Tsize_t { return c.Xyyjson_go_mut_get_len(tls, uintptr(val)) }))
}

func MutValEqualsString(val MutVal, s string) bool {
	tls := libc.NewTLS()
	defer tls.Close()
	ptr, err := allocBytes(tls, []byte(s))
	if err != nil {
		return false
	}
	defer libc.Xfree(tls, ptr)
	return c.Xyyjson_go_mut_equals_strn(tls, uintptr(val), ptr, c.Tsize_t(len(s))) != 0
}

func MutValEqual(lhs, rhs MutVal) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_mut_equals(tls, uintptr(lhs), uintptr(rhs))
	}) != 0
}

func MutArrLen(arr MutVal) uint64 {
	return uint64(withTLSSize(func(tls *libc.TLS) c.Tsize_t { return c.Xyyjson_go_mut_arr_size(tls, uintptr(arr)) }))
}

func MutArrGet(arr MutVal, idx uint64) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_mut_arr_get(tls, uintptr(arr), c.Tsize_t(idx))
	}))
}

func MutArrAppend(arr, val MutVal) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_mut_arr_append(tls, uintptr(arr), uintptr(val))
	}) != 0
}

func MutArrInsert(arr MutVal, idx uint64, val MutVal) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_mut_arr_insert(tls, uintptr(arr), uintptr(val), c.Tsize_t(idx))
	}) != 0
}

func MutArrReplace(arr MutVal, idx uint64, val MutVal) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_mut_arr_replace(tls, uintptr(arr), c.Tsize_t(idx), uintptr(val))
	}))
}

func MutArrRemove(arr MutVal, idx uint64) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_go_mut_arr_remove(tls, uintptr(arr), c.Tsize_t(idx))
	}))
}

func MutArrClear(arr MutVal) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_mut_arr_clear(tls, uintptr(arr))
	}) != 0
}

func MutArrRotate(arr MutVal, idx uint64) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_mut_arr_rotate(tls, uintptr(arr), c.Tsize_t(idx))
	}) != 0
}

func MutArrValues(arr MutVal) []MutVal {
	tls := libc.NewTLS()
	defer tls.Close()
	var iter c.Tyyjson_mut_arr_iter
	if c.Xyyjson_go_mut_arr_iter_init(tls, uintptr(arr), uintptr(unsafe.Pointer(&iter))) == 0 {
		return nil
	}
	values := make([]MutVal, 0, int(iter.Fmax))
	for {
		val := c.Xyyjson_go_mut_arr_iter_next(tls, uintptr(unsafe.Pointer(&iter)))
		if val == 0 {
			return values
		}
		values = append(values, MutVal(val))
	}
}

func MutObjLen(obj MutVal) uint64 {
	return uint64(withTLSSize(func(tls *libc.TLS) c.Tsize_t { return c.Xyyjson_go_mut_obj_size(tls, uintptr(obj)) }))
}

func MutObjGet(obj MutVal, key string) MutVal {
	tls := libc.NewTLS()
	defer tls.Close()
	ptr, err := allocBytes(tls, []byte(key))
	if err != nil {
		return 0
	}
	defer libc.Xfree(tls, ptr)
	return MutVal(c.Xyyjson_go_mut_obj_getn(tls, uintptr(obj), ptr, c.Tsize_t(len(key))))
}

func MutObjAdd(obj, key, val MutVal) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_mut_obj_add(tls, uintptr(obj), uintptr(key), uintptr(val))
	}) != 0
}

func MutObjPut(obj, key, val MutVal) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_mut_obj_put(tls, uintptr(obj), uintptr(key), uintptr(val))
	}) != 0
}

func MutObjRemoveKey(obj MutVal, key string) MutVal {
	tls := libc.NewTLS()
	defer tls.Close()
	ptr, err := allocBytes(tls, []byte(key))
	if err != nil {
		return 0
	}
	defer libc.Xfree(tls, ptr)
	return MutVal(c.Xyyjson_go_mut_obj_remove_keyn(tls, uintptr(obj), ptr, c.Tsize_t(len(key))))
}

func MutObjClear(obj MutVal) bool {
	return withTLSUint8(func(tls *libc.TLS) uint8 {
		return c.Xyyjson_go_mut_obj_clear(tls, uintptr(obj))
	}) != 0
}

func MutObjRenameKey(doc MutDoc, obj MutVal, key, newKey string) bool {
	tls := libc.NewTLS()
	defer tls.Close()
	oldPtr, err := allocBytes(tls, []byte(key))
	if err != nil {
		return false
	}
	defer libc.Xfree(tls, oldPtr)
	newPtr, err := allocBytes(tls, []byte(newKey))
	if err != nil {
		return false
	}
	defer libc.Xfree(tls, newPtr)
	return c.Xyyjson_go_mut_obj_rename_keyn(tls, uintptr(doc), uintptr(obj), oldPtr, c.Tsize_t(len(key)), newPtr, c.Tsize_t(len(newKey))) != 0
}

func MutObjMembers(obj MutVal) []MutMember {
	tls := libc.NewTLS()
	defer tls.Close()
	var iter c.Tyyjson_mut_obj_iter
	if c.Xyyjson_go_mut_obj_iter_init(tls, uintptr(obj), uintptr(unsafe.Pointer(&iter))) == 0 {
		return nil
	}
	members := make([]MutMember, 0, int(iter.Fmax))
	for {
		key := c.Xyyjson_go_mut_obj_iter_next(tls, uintptr(unsafe.Pointer(&iter)))
		if key == 0 {
			return members
		}
		val := c.Xyyjson_go_mut_obj_iter_get_val(tls, key)
		members = append(members, MutMember{Key: MutVal(key), Val: MutVal(val)})
	}
}

type MutMember struct {
	Key MutVal
	Val MutVal
}

func MutDocPointer(doc MutDoc, ptr string) (MutVal, *PointerError) {
	tls := libc.NewTLS()
	defer tls.Close()
	cptr, err := allocBytes(tls, []byte(ptr))
	if err != nil {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	defer libc.Xfree(tls, cptr)
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_ptr_err{})))
	if errp == 0 {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: "allocate pointer error"}
	}
	defer libc.Xfree(tls, errp)
	out := c.Xyyjson_go_mut_doc_ptr_getx(tls, uintptr(doc), cptr, c.Tsize_t(len(ptr)), 0, errp)
	if out == 0 {
		return 0, pointerError(pointerErrorAt(errp))
	}
	return MutVal(out), nil
}

func MutDocPointerSet(doc MutDoc, ptr string, val MutVal, createParent bool) *PointerError {
	return mutDocPointerBool(doc, ptr, val, createParent, c.Xyyjson_go_mut_doc_ptr_setx)
}

func MutDocPointerAdd(doc MutDoc, ptr string, val MutVal, createParent bool) *PointerError {
	return mutDocPointerBool(doc, ptr, val, createParent, c.Xyyjson_go_mut_doc_ptr_addx)
}

func MutDocPointerReplace(doc MutDoc, ptr string, val MutVal) (MutVal, *PointerError) {
	return mutDocPointerVal(doc, ptr, val, c.Xyyjson_go_mut_doc_ptr_replacex)
}

func MutDocPointerRemove(doc MutDoc, ptr string) (MutVal, *PointerError) {
	tls := libc.NewTLS()
	defer tls.Close()
	cptr, err := allocBytes(tls, []byte(ptr))
	if err != nil {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	defer libc.Xfree(tls, cptr)
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_ptr_err{})))
	if errp == 0 {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: "allocate pointer error"}
	}
	defer libc.Xfree(tls, errp)
	out := c.Xyyjson_go_mut_doc_ptr_removex(tls, uintptr(doc), cptr, c.Tsize_t(len(ptr)), 0, errp)
	if out == 0 {
		return 0, pointerError(pointerErrorAt(errp))
	}
	return MutVal(out), nil
}

func Patch(doc MutDoc, orig Val, patch Val) (MutVal, *PatchError) {
	tls := libc.NewTLS()
	defer tls.Close()
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_patch_err{})))
	if errp == 0 {
		return 0, &PatchError{Code: uint32(c.YYJSON_PATCH_ERROR_MEMORY_ALLOCATION), Message: "allocate patch error"}
	}
	defer libc.Xfree(tls, errp)
	out := c.Xyyjson_patch(tls, uintptr(doc), uintptr(orig), uintptr(patch), errp)
	if out == 0 {
		return 0, patchError(patchErrorAt(errp))
	}
	return MutVal(out), nil
}

func MergePatch(doc MutDoc, orig Val, patch Val) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr {
		return c.Xyyjson_merge_patch(tls, uintptr(doc), uintptr(orig), uintptr(patch))
	}))
}

func readError(err c.Tyyjson_read_err) *ReadError {
	return &ReadError{
		Code:     uint32(err.Fcode),
		Message:  libc.GoString(err.Fmsg),
		Position: uint64(err.Fpos),
	}
}

func writeError(err c.Tyyjson_write_err) *WriteError {
	return &WriteError{Code: uint32(err.Fcode), Message: libc.GoString(err.Fmsg)}
}

func pointerError(err c.Tyyjson_ptr_err) *PointerError {
	return &PointerError{Code: uint32(err.Fcode), Message: libc.GoString(err.Fmsg), Position: uint64(err.Fpos)}
}

func patchError(err c.Tyyjson_patch_err) *PatchError {
	return &PatchError{
		Code:    uint32(err.Fcode),
		Index:   uint64(err.Fidx),
		Message: libc.GoString(err.Fmsg),
		Pointer: PointerError{
			Code:     uint32(err.Fptr.Fcode),
			Message:  libc.GoString(err.Fptr.Fmsg),
			Position: uint64(err.Fptr.Fpos),
		},
	}
}

func write(fn func(*libc.TLS, uintptr, uintptr) uintptr) ([]byte, *WriteError) {
	tls := libc.NewTLS()
	defer tls.Close()
	lenp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tsize_t(0))))
	if lenp == 0 {
		return nil, &WriteError{Code: uint32(c.YYJSON_WRITE_ERROR_MEMORY_ALLOCATION), Message: "allocate write length"}
	}
	defer libc.Xfree(tls, lenp)
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_write_err{})))
	if errp == 0 {
		return nil, &WriteError{Code: uint32(c.YYJSON_WRITE_ERROR_MEMORY_ALLOCATION), Message: "allocate write error"}
	}
	defer libc.Xfree(tls, errp)
	out := fn(tls, lenp, errp)
	if out == 0 {
		return nil, writeError(writeErrorAt(errp))
	}
	defer libc.Xfree(tls, out)
	outLen := sizeAt(lenp)
	if outLen > c.Tsize_t(math.MaxInt) {
		return nil, &WriteError{Code: uint32(c.YYJSON_WRITE_ERROR_MEMORY_ALLOCATION), Message: "write output is too large"}
	}
	return append([]byte(nil), cBytes(out, int(outLen))...), nil
}

func writeFile(path string, fn func(*libc.TLS, uintptr, uintptr) uint8) *WriteError {
	if hasNULByte(path) {
		return &WriteError{Code: uint32(c.YYJSON_WRITE_ERROR_INVALID_PARAMETER), Message: "input path contains NUL byte"}
	}

	tls := libc.NewTLS()
	defer tls.Close()
	cpath, err := libc.CString(path)
	if err != nil {
		return &WriteError{Code: uint32(c.YYJSON_WRITE_ERROR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	defer libc.Xfree(tls, cpath)
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_write_err{})))
	if errp == 0 {
		return &WriteError{Code: uint32(c.YYJSON_WRITE_ERROR_MEMORY_ALLOCATION), Message: "allocate write error"}
	}
	defer libc.Xfree(tls, errp)
	if fn(tls, cpath, errp) == 0 {
		return writeError(writeErrorAt(errp))
	}
	return nil
}

func allocBytes(tls *libc.TLS, data []byte) (uintptr, error) {
	if len(data) > math.MaxInt-readPaddingSize {
		return 0, fmt.Errorf("allocate input buffer: size overflow")
	}
	size := len(data) + readPaddingSize
	ptr := libc.Xmalloc(tls, c.Tsize_t(size))
	if ptr == 0 {
		return 0, fmt.Errorf("allocate %d bytes", size)
	}
	buf := cBytes(ptr, size)
	copy(buf, data)
	clear(buf[len(data):])
	return ptr, nil
}

func hasNULByte(s string) bool {
	return strings.IndexByte(s, 0) >= 0
}

func copyCStringLen(ptr uintptr, n int) string {
	if ptr == 0 || n <= 0 {
		return ""
	}
	return string(cBytes(ptr, n))
}

// cPointer converts modernc/libc addresses while the owning TLS allocation is live.
//
//nolint:govet // modernc.org/libc exposes C pointers as uintptr.
//go:nocheckptr
func cPointer(addr uintptr) unsafe.Pointer {
	return unsafe.Pointer(addr)
}

//go:nocheckptr
func cBytes(addr uintptr, n int) []byte {
	return unsafe.Slice((*byte)(cPointer(addr)), n)
}

//go:nocheckptr
func cBytesAt(addr uintptr, off, n int) []byte {
	return unsafe.Slice((*byte)(unsafe.Add(cPointer(addr), off)), n)
}

//go:nocheckptr
func sizeAt(addr uintptr) c.Tsize_t {
	return *(*c.Tsize_t)(cPointer(addr))
}

//go:nocheckptr
func readErrorAt(addr uintptr) c.Tyyjson_read_err {
	return *(*c.Tyyjson_read_err)(cPointer(addr))
}

//go:nocheckptr
func writeErrorAt(addr uintptr) c.Tyyjson_write_err {
	return *(*c.Tyyjson_write_err)(cPointer(addr))
}

//go:nocheckptr
func pointerErrorAt(addr uintptr) c.Tyyjson_ptr_err {
	return *(*c.Tyyjson_ptr_err)(cPointer(addr))
}

//go:nocheckptr
func patchErrorAt(addr uintptr) c.Tyyjson_patch_err {
	return *(*c.Tyyjson_patch_err)(cPointer(addr))
}

func withTLSUintptr(fn func(*libc.TLS) uintptr) uintptr {
	tls := libc.NewTLS()
	defer tls.Close()
	return fn(tls)
}

func withTLSUint8(fn func(*libc.TLS) uint8) uint8 {
	tls := libc.NewTLS()
	defer tls.Close()
	return fn(tls)
}

func withTLSSize(fn func(*libc.TLS) c.Tsize_t) c.Tsize_t {
	tls := libc.NewTLS()
	defer tls.Close()
	return fn(tls)
}

func withTLSUint64(fn func(*libc.TLS) c.Tuint64_t) c.Tuint64_t {
	tls := libc.NewTLS()
	defer tls.Close()
	return fn(tls)
}

func withTLSInt64(fn func(*libc.TLS) c.Tint64_t) c.Tint64_t {
	tls := libc.NewTLS()
	defer tls.Close()
	return fn(tls)
}

func withTLSFloat64(fn func(*libc.TLS) float64) float64 {
	tls := libc.NewTLS()
	defer tls.Close()
	return fn(tls)
}

func withTLSString(fn func(*libc.TLS) uintptr) string {
	tls := libc.NewTLS()
	defer tls.Close()
	return libc.GoString(fn(tls))
}

func mutNew(doc MutDoc, fn func(*libc.TLS, uintptr) uintptr) MutVal {
	return MutVal(withTLSUintptr(func(tls *libc.TLS) uintptr { return fn(tls, uintptr(doc)) }))
}

func mutDocPointerBool(doc MutDoc, ptr string, val MutVal, createParent bool, fn func(*libc.TLS, uintptr, uintptr, c.Tsize_t, uintptr, uint8, uintptr, uintptr) uint8) *PointerError {
	tls := libc.NewTLS()
	defer tls.Close()
	cptr, err := allocBytes(tls, []byte(ptr))
	if err != nil {
		return &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	defer libc.Xfree(tls, cptr)
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_ptr_err{})))
	if errp == 0 {
		return &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: "allocate pointer error"}
	}
	defer libc.Xfree(tls, errp)
	if fn(tls, uintptr(doc), cptr, c.Tsize_t(len(ptr)), uintptr(val), boolByte(createParent), 0, errp) == 0 {
		return pointerError(pointerErrorAt(errp))
	}
	return nil
}

func mutDocPointerVal(doc MutDoc, ptr string, val MutVal, fn func(*libc.TLS, uintptr, uintptr, c.Tsize_t, uintptr, uintptr, uintptr) uintptr) (MutVal, *PointerError) {
	tls := libc.NewTLS()
	defer tls.Close()
	cptr, err := allocBytes(tls, []byte(ptr))
	if err != nil {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: err.Error()}
	}
	defer libc.Xfree(tls, cptr)
	errp := libc.Xmalloc(tls, c.Tsize_t(unsafe.Sizeof(c.Tyyjson_ptr_err{})))
	if errp == 0 {
		return 0, &PointerError{Code: uint32(c.YYJSON_PTR_ERR_MEMORY_ALLOCATION), Message: "allocate pointer error"}
	}
	defer libc.Xfree(tls, errp)
	out := fn(tls, uintptr(doc), cptr, c.Tsize_t(len(ptr)), uintptr(val), 0, errp)
	if out == 0 {
		return 0, pointerError(pointerErrorAt(errp))
	}
	return MutVal(out), nil
}

func boolByte(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}
