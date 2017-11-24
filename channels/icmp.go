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
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"os"
)

const (
	ProtocolICMP   = 1 /* from iana.ProtocolICMP which being internal I can't use -.- */
	ICMPChunkSize  = 128
	ICMPBufferSize = 512
)

type ICMPChannel struct {
	is_client bool
	address   string
	seqn      uint32
	conn      *icmp.PacketConn
	chunks    chan []byte
	stats     Stats
}

func NewICMPChannel() *ICMPChannel {
	return &ICMPChannel{
		is_client: true,
		address:   "0.0.0.0",
		seqn:      0,
		conn:      nil,
		chunks:    make(chan []byte),
	}
}

func (c *ICMPChannel) Name() string {
	return "icmp"
}

func (c *ICMPChannel) Description() string {
	return "Send data as ICMP packets and read data as ICMP packets ( example: icmp:192.168.1.24 or just icmp for 0.0.0.0 )."
}

func (c *ICMPChannel) Register() error {
	return nil
}

func (c *ICMPChannel) Setup(direction Direction, args string) (err error) {
	if direction == INPUT_CHANNEL {
		c.is_client = false

	} else {
		c.is_client = true
	}

	if args != "" {
		c.address = args
	}
	return nil
}

func (c *ICMPChannel) Start() (err error) {
	if c.is_client == true {
		if c.conn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0"); err != nil {
			return err
		}
	} else {
		if c.conn, err = icmp.ListenPacket("ip4:icmp", c.address); err != nil {
			return err
		}

		go func() {
			sg1.Log("Started ICMP listener on %s ...\n\n", c.address)

			defer c.conn.Close()

			buffer := make([]byte, ICMPBufferSize)

			for {
				n, peer, err := c.conn.ReadFrom(buffer)
				if err != nil {
					sg1.Log("Error while reading ICMP packet: %s.\n", err)
					continue
				}

				// sg1.Log("Read %d bytes of ICMP packet from %s .\n", n, peer)

				msg, err := icmp.ParseMessage(ProtocolICMP, buffer[:n])
				if err != nil {
					sg1.Log("Error while parsing ICMP packet sent by %s: %s.\n", peer, err)
					continue
				}

				if msg.Type == ipv4.ICMPTypeEcho {
					echo := msg.Body.(*icmp.Echo)
					if packet, err := DecodePacket(echo.Data); err == nil {
						c.stats.TotalRead += int(packet.DataSize)
						c.chunks <- packet.Data
					} else {
						sg1.Log("Error while decoding ICMP payload: %s.\n", err)
					}
				}
			}
		}()
	}

	return nil
}

func (c *ICMPChannel) HasReader() bool {
	if c.is_client {
		return false
	}
	return true
}

func (c *ICMPChannel) HasWriter() bool {
	if c.is_client {
		return true
	}
	return false
}

func (c *ICMPChannel) Read(b []byte) (n int, err error) {
	if c.is_client {
		return 0, fmt.Errorf("icmp client can't be used for reading.")
	}

	data := <-c.chunks
	for i, c := range data {
		b[i] = c
	}

	return len(data), nil
}

func (c *ICMPChannel) sendPacket(packet *Packet) error {
	data := packet.Raw()
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: int(c.seqn),
			Data: data,
		},
	}

	raw, err := msg.Marshal(nil)
	if err != nil {
		return err
	}

	if _, err := c.conn.WriteTo(raw, &net.IPAddr{IP: net.ParseIP(c.address)}); err != nil {
		return err
	}

	return nil
}

func (c *ICMPChannel) Write(b []byte) (n int, err error) {
	if c.is_client == false {
		return 0, fmt.Errorf("icmp server can't be used for writing.")
	}

	wrote := 0
	for _, chunk := range BufferToChunks(b, ICMPChunkSize) {
		size := len(chunk)
		packet := NewPacket(c.seqn, uint32(size), chunk)

		if err := c.sendPacket(packet); err != nil {
			sg1.Log("%s\n", err)
		} else {
			wrote += size
			c.stats.TotalWrote += size
		}

		c.seqn++
	}

	return wrote, nil
}

func (c *ICMPChannel) Stats() Stats {
	return c.stats
}
