package prql

// #cgo CFLAGS: -I.
// #cgo darwin,arm64 LDFLAGS: -L${SRCDIR} -lprqlc-aarch64-apple-darwin
// #cgo darwin,amd64 LDFLAGS: -L${SRCDIR} -lprqlc-x86_64-apple-darwin
// #cgo linux,arm64 LDFLAGS: -L${SRCDIR} -lprqlc-aarch64-unknown-linux-musl
// #cgo linux,amd64 LDFLAGS: -L${SRCDIR} -lprqlc-x86_64-unknown-linux-musl -lm
// #cgo windows,amd64 LDFLAGS: -L${SRCDIR} -lprqlc-x86_64-pc-windows-gnu -lws2_32 -luserenv -lntdll
/*#include <stdlib.h>
#include "prqlc.h"

Options global_options = {
	false, // Remove the pretty print
	"sql.sqlite", // The dialect to use
	false // Remove the trailing comments
};

CompileResult to_sql(char *prql_query) {
	return compile(prql_query, &global_options);
}

*/
import "C"

import (
	"unsafe"
)

type SourceLocationError struct {
	startLine   int
	startColumn int
	endLine     int
	endColumn   int
}

type CompileMessage struct {
	ErrorCode rune
	// Annoted code containing the error and the hints
	Display string
	// The location of the error
	LocationError SourceLocationError
}

func ToSQL(prqlQuery string) (string, []CompileMessage) {
	res := C.to_sql(C.CString(prqlQuery))
	a := res.messages
	if res.messages_len > 0 {
		messages := make([]CompileMessage, res.messages_len)
		// Convert the messages
		for i := 0; i < int(res.messages_len); i++ {
			message := (*C.struct_Message)(unsafe.Pointer(uintptr(unsafe.Pointer(a)) + uintptr(i)*unsafe.Sizeof(*res.messages)))
			compileMessage := CompileMessage{}
			if message.code != nil && *message.code != nil {
				compileMessage.ErrorCode = rune(**message.code)
			}

			if message.display != nil && *message.display != nil {
				compileMessage.Display = C.GoString(*message.display)
			}

			if message.location != nil {
				compileMessage.LocationError.startLine = int(message.location.start_line)
				compileMessage.LocationError.startColumn = int(message.location.start_col)
				compileMessage.LocationError.endLine = int(message.location.end_line)
				compileMessage.LocationError.endColumn = int(message.location.end_col)
			}

			messages[i] = compileMessage

		}

		return "", messages
	}
	return C.GoString(res.output), nil
}
