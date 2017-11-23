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
	DataSize  uint32
	Data      []byte
}

func NewPacket(seqn uint32, datasize uint32, data []byte) *Packet {
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
	size_buf := buffer[4:8]
	// sg1.Log("seqn_buf=%s\nsize_buf=%s\n", hex.EncodeToString(seqn_buf), hex.EncodeToString(size_buf))
	max_size := len(buffer) - p.HeaderSize()

	// TODO: Check sequence number
	seqn := binary.BigEndian.Uint32(seqn_buf)
	size := binary.BigEndian.Uint32(size_buf)

	data_buf := make([]byte, 0)
	if size > uint32(max_size) {
		return nil, fmt.Errorf("Data size %d is more than the %d bytes of available payload.", size, max_size)
	} else if size > 0 {
		data_buf = buffer[8:]
	}

	size = uint32(len(data_buf))

	return NewPacket(seqn, size, data_buf), nil
}

func (p *Packet) HeaderSize() int {
	return 4 + 4
}

func (p *Packet) Encode() []byte {
	seqn_buf := make([]byte, 4)
	size_buf := make([]byte, 4)

	binary.BigEndian.PutUint32(seqn_buf, p.SeqNumber)
	binary.BigEndian.PutUint32(seqn_buf, p.DataSize)

	buffer := append(seqn_buf, p.Data...)
	buffer = append(size_buf, buffer...)

	return buffer
}

func PadBuffer(buf []byte, size int, pad byte) []byte {
	buf_sz := len(buf)
	if buf_sz < size {
		to_pad := size - buf_sz
		for i := 0; i < to_pad; i++ {
			buf = append(buf, pad)
		}
	}

	return buf
}

func BufferToChunks(buffer []byte, chunk_size int) [][]byte {
	total_sz := len(buffer)
	left := total_sz
	done := 0
	chunks := make([][]byte, 0)

	for left > 0 {
		size := chunk_size
		if left < size {
			size = left
		}

		chunk := buffer[done : done+size]
		chunk = PadBuffer(chunk, chunk_size, 0x00)

		chunks = append(chunks, chunk)

		done += size
		left -= size
	}

	return chunks
}
