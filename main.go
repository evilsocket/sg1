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
	"strings"
	"time"

	"github.com/evilsocket/sg1/channels"
	"github.com/evilsocket/sg1/modules"
	"github.com/evilsocket/sg1/sg1"
)

func init() {
	flag.StringVar(&sg1.From, "in", sg1.From, "Read input data from this channel.")
	flag.StringVar(&sg1.To, "out", sg1.To, "Write output data to this channel.")
	flag.StringVar(&sg1.ModuleNames, "modules", sg1.ModuleNames, "Comma separated list of modules to use.")
	flag.IntVar(&sg1.Delay, "delay", sg1.Delay, "Delay in milliseconds to wait between one I/O loop and another, or 0 for no delay.")
	flag.IntVar(&sg1.BufferSize, "buffer-size", sg1.BufferSize, "Buffer size to use while reading data to input and writing to output.")
	flag.BoolVar(&sg1.DebugMessages, "debug", sg1.DebugMessages, "Enable debug messages.")

	channels.Register(channels.NewConsoleChannel())
	channels.Register(channels.NewTCPChannel())
	channels.Register(channels.NewUDPChannel())
	channels.Register(channels.NewTLSChannel())
	channels.Register(channels.NewDNSChannel())
	channels.Register(channels.NewICMPChannel())
	channels.Register(channels.NewPastebinChannel())

	modules.Register(modules.NewRaw())
	modules.Register(modules.NewBase64())
	modules.Register(modules.NewAES())
	modules.Register(modules.NewExec())

	flag.Usage = func() {
		// TODO: Modules and channels specific options should be grouped instead of
		// being described in the general usage section.

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
	sg1.Error("%s\n", err)
	sg1.Raw("\n")
	os.Exit(1)
}

type DataHandler func(buff []byte) (int, []byte, error)

func ReadLoop(input, output channels.Channel, buffer_size, delay int, dataHandler DataHandler) error {
	var n int
	var err error

	for {
		buff := make([]byte, buffer_size)
		// read buffer_size bytes from the input channel
		if n, err = input.Read(buff); err != nil {
			if err.Error() == "EOF" {
				break
			} else {
				return err
			}
		}

		// do we have data?
		if len(buff) > 0 && n > 0 {
			// if a handler was given, process those bytes with it
			if dataHandler != nil {
				n, buff, err = dataHandler(buff[:n])
				if err != nil {
					return err
				}
			}

			// write bytes to the output channel
			if _, err = output.Write(buff[:n]); err != nil {
				return err
			}

			// throttle if delay was specified
			if delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}
	}

	return nil
}

func main() {
	sg1.Raw(sg1.Bold("%s v%s ( %s %s )\n\n"), sg1.APP_NAME, sg1.APP_VERSION, runtime.GOOS, runtime.GOARCH)

	flag.Parse()

	var input channels.Channel
	var output channels.Channel
	var run_modules = make([]modules.Module, 0)
	var err error

	if input, err = channels.Factory(sg1.From, channels.INPUT_CHANNEL); err != nil {
		onError(err)
	}

	if output, err = channels.Factory(sg1.To, channels.OUTPUT_CHANNEL); err != nil {
		onError(err)
	}

	module_names := strings.Split(sg1.ModuleNames, ",")
	for _, name := range module_names {
		if module, err := modules.Factory(name); err != nil {
			onError(err)
		} else {
			sg1.Debug("Loaded module %s.\n", module.Name())
			run_modules = append(run_modules, module)
		}
	}

	if len(module_names) == 1 && module_names[0] == "raw" {
		sg1.Log("%s --> %s\n", input.Name(), output.Name())
	} else {
		sg1.Log("%s --> [%s] --> %s\n", input.Name(), sg1.ModuleNames, output.Name())
	}

	if err = input.Start(); err != nil {
		onError(err)
	}

	if err = output.Start(); err != nil {
		onError(err)
	}

	start := time.Now()

	err = ReadLoop(input, output, sg1.BufferSize, sg1.Delay, func(buff []byte) (int, []byte, error) {
		var run_error error
		var ret []byte

		for _, module := range run_modules {
			sg1.Debug("Running module %s on buffer of %d bytes.\n", module.Name(), len(buff))
			_, ret, run_error = module.Run(buff)
			if run_error != nil {
				sg1.Debug("run_error = %s\n", run_error)
				break
			}
			buff = ret
		}

		return len(buff), buff, run_error
	})

	if err != nil {
		sg1.Error("%s.\n", err)
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

		sg1.Raw("\n\n")
		sg1.Raw("Total read    : %s\n", sg1.FormatBytes(read))
		sg1.Raw("Total written : %s\n", sg1.FormatBytes(wrote))
		sg1.Raw("Time elapsed  : %s\n", elapsed)
		sg1.Raw("Speed         : %s\n", sg1.FormatSpeed(bps))
		sg1.Raw("\n")
	}
}
