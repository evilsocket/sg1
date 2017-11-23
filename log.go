package sg1

import (
	"fmt"
	"os"
)

func Log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}
