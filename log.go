package sg1

import (
	"encoding/hex"
	"fmt"
	"os"
)

// https://misc.flogisoft.com/bash/tip_colors_and_formatting
const (
	BOLD = "\033[1m"
	DIM  = "\033[2m"

	RED    = "\033[31m"
	GREEN  = "\033[32m"
	YELLOW = "\033[33m"

	RESET = "\033[0m"
)

func Hex(buffer []byte) string {
	return hex.EncodeToString(buffer)
}

// TODO: This should be made thread safe using a mutex in order to
// avoid text overlapping on stderr.
func Raw(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func Log(format string, args ...interface{}) {
	Raw("[INF] "+format, args...)
}

func Warning(format string, args ...interface{}) {
	Raw(YELLOW+"[WAR] "+format+RESET, args...)
}

func Error(format string, args ...interface{}) {
	Raw(RED+"[ERR] "+format+RESET, args...)
}

func Debug(format string, args ...interface{}) {
	if DebugMessages {
		Raw(DIM+"[DBG] "+format+RESET, args...)
	}
}
