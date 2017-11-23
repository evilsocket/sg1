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
package modules

import (
	"fmt"
	"github.com/evilsocket/sg1/channels"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Exec struct {
}

func NewExec() *Exec {
	return &Exec{}
}

func (m *Exec) Name() string {
	return "exec"
}

func (m *Exec) Description() string {
	return "Get command from input channel, execute and write output to output channel."
}

func (m *Exec) Register() error {
	return nil
}

func (m *Exec) Run(input, output channels.Channel, buffer_size, delay int) error {
	var n int
	var err error
	var buff = make([]byte, buffer_size)

	for {
		if n, err = input.Read(buff); err != nil {
			if err.Error() == "EOF" {
				break
			} else {
				return err
			}
		}

		var cmdout []byte

		cmdline := string(buff[:n])
		cmdline = strings.Trim(cmdline, " \t\r\n")

		// fmt.Printf("Parsing and executing command line (%d bytes) '%s'.\n", n, cmdline)

		cmd := ""
		args := []string{}

		if cmdline != "" {
			parts := strings.Fields(cmdline)
			cmd = parts[0]
			args = parts[1:len(parts)]
		}

		path, err := exec.LookPath(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while looking path of '%s': %s.\n", cmd, err)
			cmdout = []byte(err.Error())
		} else {
			raw, err := exec.Command(path, args...).CombinedOutput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while executing '%s %s': %s.\n", path, args, err)
				cmdout = []byte(err.Error())
			} else {
				cmdout = []byte(raw)
			}

			if _, err = output.Write(cmdout); err != nil {
				return err
			}
		}

		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}

	return nil
}
