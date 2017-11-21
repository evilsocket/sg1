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
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/evilsocket/sg1/channels"
	"github.com/evilsocket/sg1/modules"
)

var (
	from        = "stdin"
	to          = "stdout"
	module_name = ""
)

func init() {
	flag.StringVar(&from, "in", "stdin", "Read input data from this channel.")
	flag.StringVar(&to, "out", "stdout", "Write output data to this channel.")
	flag.StringVar(&module_name, "module", "raw", "Module name to use.")

	channels.Register(channels.NewSTDINChannel())
	channels.Register(channels.NewSTDOUTChannel())
	channels.Register(channels.NewTCPClientChannel())
	channels.Register(channels.NewTCPServerChannel())
	channels.Register(channels.NewDNSClientChannel())
	channels.Register(channels.NewDNSServerChannel())

	modules.Register(modules.NewRaw())
	modules.Register(modules.NewAES())

	flag.Usage = func() {
		fmt.Printf("Usage of sg1:\n\n")
		flag.PrintDefaults()

		fmt.Println()
		fmt.Printf("Available modules:\n\n")

		for name, module := range modules.Registered() {
			fmt.Printf("  %10s : %s\n", name, module.Description())
		}
		fmt.Println()
		fmt.Printf("Available channels:\n\n")

		for name, channel := range channels.Registered() {
			fmt.Printf("  %10s : %s\n", name, channel.Description())
		}

		fmt.Println()
	}
}

func onError(err error) {
	fmt.Println(err)
	fmt.Println()
	// flag.Usage()
	os.Exit(1)
}

const (
	KB = 1024
	MB = 1024 * 1024
	GB = 1024 * 1024 * 1024
)

func formatBytes(bytes int) string {
	if bytes < KB {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < MB {
		return fmt.Sprintf("%d KB", bytes/KB)
	} else if bytes < GB {
		return fmt.Sprintf("%d MB", bytes/MB)
	} else {
		return fmt.Sprintf("%d GB", bytes/GB)
	}
}

func formatSpeed(bps float64) string {
	return fmt.Sprintf("%s/s", formatBytes(int(bps)))
}

func main() {
	fmt.Fprintf(os.Stderr, "%s v%s ( built on %s for %s %s )\n\n", APP_NAME, APP_VERSION, APP_BUILD_DATE, runtime.GOOS, runtime.GOARCH)

	flag.Parse()

	var input channels.Channel
	var output channels.Channel
	var module modules.Module
	var err error

	if input, err = channels.Factory(from, true); err != nil {
		onError(err)
	}

	if output, err = channels.Factory(to, false); err != nil {
		onError(err)
	}

	if module, err = modules.Factory(module_name); err != nil {
		onError(err)
	}

	fmt.Fprintf(os.Stderr, "  %s --> [%s] --> %s\n\n", input.Name(), module.Name(), output.Name())

	if err = input.Start(); err != nil {
		onError(err)
	}

	if err = output.Start(); err != nil {
		onError(err)
	}

	start := time.Now()

	if err = module.Run(input, output); err != nil {
		fmt.Println(err)
	} else {
		elapsed := time.Since(start)
		es := elapsed.Seconds()
		bps := float64(0.0)
		read := input.Stats().TotalRead
		wrote := output.Stats().TotalWrote

		if read < wrote {
			bps = float64(read) / es
		} else {
			bps = float64(wrote) / es
		}

		fmt.Fprintf(os.Stderr, "\n\n")
		fmt.Fprintf(os.Stderr, "Total read    : %s\n", formatBytes(read))
		fmt.Fprintf(os.Stderr, "Total written : %s\n", formatBytes(wrote))
		fmt.Fprintf(os.Stderr, "Time elapsed  : %s\n", elapsed)
		fmt.Fprintf(os.Stderr, "Speed         : %s\n", formatSpeed(bps))
	}
}
