package sg1

import (
	"encoding/hex"
	"fmt"
	"os"
	"sync"
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

var (
	logLock = &sync.Mutex{}
)

func Hex(buffer []byte) string {
	return hex.EncodeToString(buffer)
}

func Raw(format string, args ...interface{}) {
	logLock.Lock()
	defer logLock.Unlock()

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
