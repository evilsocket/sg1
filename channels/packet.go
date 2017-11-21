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
package channels

import (
	"encoding/binary"
	"fmt"
)

type Packet struct {
	SeqNumber uint32
	DataSize  uint8
	Data      []byte
}

func NewPacket(seqn uint32, datasize uint8, data []byte) *Packet {
	return &Packet{
		SeqNumber: seqn,
		DataSize:  datasize,
		Data:      data,
	}
}

func DecodePacket(buffer []byte) (p *Packet, err error) {
	buf_size := len(buffer)
	if buf_size < p.HeaderSize() {
		return nil, fmt.Errorf("Buffer size %d is less than minimum required.", buf_size)
	}

	seqn_buf := buffer[0:4]
	size_buf := buffer[4:5]
	max_size := len(buffer) - p.HeaderSize()

	// TODO: Check sequence number
	seqn := binary.BigEndian.Uint32(seqn_buf)

	data_buf := make([]byte, 0)
	size := uint8(size_buf[0])
	if int(size) > max_size {
		return nil, fmt.Errorf("Data size %d is more than the %d bytes of available payload.", size, max_size)
	} else if size > 0 {
		data_buf = buffer[5:]
	}

	return NewPacket(seqn, size, data_buf[:size]), nil
}

func (p *Packet) HeaderSize() int {
	return 4 + 1
}

func (p *Packet) Encode() []byte {
	seqn_buf := make([]byte, 4)
	size_buf := make([]byte, 1)

	binary.BigEndian.PutUint32(seqn_buf, p.SeqNumber)
	size_buf[0] = byte(p.DataSize & 0xff)

	buffer := append(size_buf, p.Data...)
	buffer = append(seqn_buf, buffer...)

	return buffer
}
