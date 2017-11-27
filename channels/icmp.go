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
	"github.com/evilsocket/sg1/sg1"
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
	seq       *sg1.PacketSequencer
	conn      *icmp.PacketConn
	stats     Stats
}

func NewICMPChannel() *ICMPChannel {
	return &ICMPChannel{
		is_client: true,
		address:   "0.0.0.0",
		seq:       sg1.NewPacketSequencer(),
		conn:      nil,
	}
}

func (c *ICMPChannel) Copy() interface{} {
	return NewICMPChannel()
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

	sg1.Debug("Setup ICMP channel: direction=%d address=%s\n", direction, c.address)

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
			defer c.conn.Close()

			sg1.Log("Started ICMP listener on %s ...\n\n", c.address)

			buffer := make([]byte, ICMPBufferSize)
			for {
				n, peer, err := c.conn.ReadFrom(buffer)
				if err != nil {
					sg1.Warning("Error while reading ICMP packet: %s.\n", err)
					continue
				}

				sg1.Debug("Read %d bytes of ICMP packet from %s .\n", n, peer)

				msg, err := icmp.ParseMessage(ProtocolICMP, buffer[:n])
				if err != nil {
					sg1.Warning("Error while parsing ICMP packet sent by %s: %s.\n", peer, err)
					continue
				}

				if msg.Type == ipv4.ICMPTypeEcho {
					sg1.Debug("Got ICMP echo.\n")
					echo := msg.Body.(*icmp.Echo)
					if packet, err := sg1.DecodePacket(echo.Data); err == nil {
						sg1.Debug("Decoded packet of %d bytes from ICMP echo payload (seqn=%d).\n", packet.DataSize, packet.SeqNumber)

						c.stats.TotalRead += int(packet.DataSize)
						c.seq.Add(packet)
					} else {
						sg1.Error("Error while decoding ICMP payload: %s.\n", err)
					}
				} else {
					sg1.Debug("ICMP packet is not an echo.\n")
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

	packet := c.seq.Get()
	data := packet.Data
	for i, c := range data {
		b[i] = c
	}

	sg1.Debug("Read %d bytes from ICMP listener.\n", len(data))

	return len(data), nil
}

func (c *ICMPChannel) sendPacket(packet *sg1.Packet) error {
	sg1.Debug("Encapsulating %d bytes of packet in ICMP echo payload for address %s.\n", packet.DataSize, c.address)

	data := packet.Raw()
	sg1.Debug("HEX: %s\n", sg1.Hex(data))
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: int(packet.SeqNumber),
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

	sg1.Debug("Writing %d bytes to ICMP channel as chunks of %d bytes.\n", len(b), ICMPChunkSize)

	wrote := 0
	for _, chunk := range sg1.BufferToChunks(b, ICMPChunkSize) {
		size := len(chunk)
		packet := c.seq.Packet(chunk)

		sg1.Debug("Sending %d bytes of encoded packet (seqn=%d).\n", packet.DataSize, packet.SeqNumber)

		if err := c.sendPacket(packet); err != nil {
			sg1.Error("Error while sending ICMP packet: %s\n", err)
		} else {
			sg1.Debug("Wrote %d bytes.\n", size)
			wrote += size
			c.stats.TotalWrote += size
		}
	}

	sg1.Debug("Wrote %d bytes to ICMP channel.\n", wrote)

	return wrote, nil
}

func (c *ICMPChannel) Stats() Stats {
	return c.stats
}
