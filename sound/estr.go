package sound

/*
#include "estr.h"
*/
import "C"
import "unsafe"

func (estr *C.EStr) String() string {
	cstr := (*C.char)(unsafe.Pointer(estr))
	return C.GoString(cstr)
}
