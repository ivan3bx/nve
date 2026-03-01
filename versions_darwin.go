//go:build darwin

package nve

/*
#cgo LDFLAGS: -framework Foundation
#include <stdlib.h>
extern int saveVersion(const char *filePath);
*/
import "C"

import (
	"log"
	"unsafe"
)

// SaveFileVersion registers the current state of the file at the given path
// as a version with macOS's native file versioning system.
func SaveFileVersion(path string) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	result := C.saveVersion(cPath)
	if result != 0 {
		log.Printf("[WARN] SaveFileVersion: failed for %s", path)
	} else {
		log.Printf("[DEBUG] SaveFileVersion: registered version for %s", path)
	}
}
