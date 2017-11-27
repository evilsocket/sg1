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
	"fmt"
	"github.com/evilsocket/sg1"
	"net"
)

const (
	UDPChunkSize  = 128
	UDPBufferSize = 512
)

type UDPChannel struct {
	is_client bool
	address   *net.UDPAddr
	seqn      uint32
	conn      *net.UDPConn
	chunks    chan []byte
	stats     Stats
}

func NewUDPChannel() *UDPChannel {
	return &UDPChannel{
		is_client: true,
		seqn:      0,
		address:   nil,
		conn:      nil,
		chunks:    make(chan []byte),
	}
}

func (c *UDPChannel) Copy() interface{} {
	return NewUDPChannel()
}

func (c *UDPChannel) Name() string {
	return "udp"
}

func (c *UDPChannel) Description() string {
	return "Send data as UDP packets and read data as UDP packets ( example: udp:192.168.1.24:10013 )."
}

func (c *UDPChannel) Register() error {
	return nil
}

func (c *UDPChannel) Setup(direction Direction, args string) (err error) {
	if direction == INPUT_CHANNEL {
		c.is_client = false

	} else {
		c.is_client = true
	}

	if c.address, err = net.ResolveUDPAddr("udp", args); err != nil {
		return err
	}

	sg1.Debug("Setup UDP channel: direction=%d address=%s\n", direction, c.address)

	return nil
}

func (c *UDPChannel) Start() (err error) {
	if c.is_client == true {
		local, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		if err != nil {
			return err
		}

		if c.conn, err = net.DialUDP("udp", local, c.address); err != nil {
			return err
		}
	} else {
		if c.conn, err = net.ListenUDP("udp", c.address); err != nil {
			return err
		}

		go func() {
			defer c.conn.Close()

			sg1.Log("Started UDP listener on %s ...\n\n", c.address)

			buffer := make([]byte, UDPBufferSize)
			for {
				n, peer, err := c.conn.ReadFrom(buffer)
				if err != nil {
					sg1.Warning("Error while reading UDP packet: %s.\n", err)
					continue
				}

				sg1.Debug("Read %d bytes of UDP packet from %s .\n", n, peer)

				if packet, err := DecodePacket(buffer[:n]); err == nil {
					sg1.Debug("Decoded packet of %d bytes from UDP echo payload.\n", packet.DataSize)

					c.stats.TotalRead += int(packet.DataSize)
					c.chunks <- packet.Data
				} else {
					sg1.Error("Error while decoding UDP payload: %s.\n", err)
				}
			}
		}()
	}

	return nil
}

func (c *UDPChannel) HasReader() bool {
	if c.is_client {
		return false
	}
	return true
}

func (c *UDPChannel) HasWriter() bool {
	if c.is_client {
		return true
	}
	return false
}

func (c *UDPChannel) Read(b []byte) (n int, err error) {
	if c.is_client {
		return 0, fmt.Errorf("icmp client can't be used for reading.")
	}

	data := <-c.chunks
	for i, c := range data {
		b[i] = c
	}

	sg1.Debug("Read %d bytes from UDP listener.\n", len(data))

	return len(data), nil
}

func (c *UDPChannel) sendPacket(packet *Packet) error {
	sg1.Debug("Encapsulating %d bytes of packet in UDP echo payload for address %s.\n", packet.DataSize, c.address)

	data := packet.Raw()
	if _, err := c.conn.Write(data); err != nil {
		return err
	}

	return nil
}

func (c *UDPChannel) Write(b []byte) (n int, err error) {
	if c.is_client == false {
		return 0, fmt.Errorf("icmp server can't be used for writing.")
	}

	sg1.Debug("Writing %d bytes to UDP channel as chunks of %d bytes.\n", len(b), UDPChunkSize)

	wrote := 0
	for _, chunk := range BufferToChunks(b, UDPChunkSize) {
		size := len(chunk)
		packet := NewPacket(c.seqn, uint32(size), chunk)

		sg1.Debug("Sending %d bytes of encoded packet.\n", packet.DataSize)

		if err := c.sendPacket(packet); err != nil {
			sg1.Error("Error while sending UDP packet: %s\n", err)
		} else {
			sg1.Debug("Wrote %d bytes.\n", size)
			wrote += size
			c.stats.TotalWrote += size
		}

		c.seqn++
	}

	sg1.Debug("Wrote %d bytes to UDP channel.\n", wrote)

	return wrote, nil
}

func (c *UDPChannel) Stats() Stats {
	return c.stats
}
