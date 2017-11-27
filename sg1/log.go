/*
* Copyleft 2017, Simone Margaritelli <evilsocket at protonmail dot com>
* Redistribution and use in source and binary forms, with or without
* modification, are permitted provided that the following conditions are met:
*
*   * Redistributions of source code must retain the above copyright notice,
*     this list of conditions and the following disclaimer.
*   * Redistributions in binary form must reproduce the above copyright
*     notice, this list of conditions and the following disclaimer in the
*     documentation and/or other materials provided with the distribution.
*   * Neither the name of ARM Inject nor the names of its contributors may be used
*     to endorse or promote products derived from this software without
*     specific prior written permission.
*
* THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
* AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
* IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
* ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
* LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
* CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
* SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
* INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
* CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
* ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
* POSSIBILITY OF SUCH DAMAGE.
 */
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

func Bold(s string) string {
	return BOLD + s + RESET
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
