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
	b64 "encoding/base64"
	"flag"
	"github.com/evilsocket/sg1/channels"
	"github.com/evilsocket/sg1/sg1"
)

type Base64 struct {
	mode string
	in   channels.Channel
	out  channels.Channel
}

func NewBase64() *Base64 {
	return &Base64{
		mode: "encode",
		in:   nil,
		out:  nil,
	}
}

func (m *Base64) Name() string {
	return "base64"
}

func (m *Base64) Description() string {
	return "Read from input, encode or decode in base64 and write to output ( use -base64-mode argument )."
}

func (m *Base64) Register() error {
	flag.StringVar(&m.mode, "base64-mode", "encode", "Base64 mode, can be 'encode' or 'decode'.")
	return nil
}

func (m *Base64) Run(buff []byte) (int, []byte, error) {
	var output []byte

	if m.mode == "encode" {
		encoded := b64.StdEncoding.EncodeToString(buff)
		output = []byte(encoded)
	} else {
		decoded, err := b64.StdEncoding.DecodeString(string(buff))
		if err != nil {
			sg1.Error("%s\n", err)
			return 0, nil, err
		}
		output = decoded
	}

	return len(output), output, nil
}
