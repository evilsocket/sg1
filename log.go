package sg1

import (
	"fmt"
	"os"
)

func Log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func Debug(format string, args ...interface{}) {
	if DebugMessages {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format, args...)
	}
}
