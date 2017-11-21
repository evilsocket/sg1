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
	"encoding/hex"
	"fmt"
	"github.com/miekg/dns"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
)

var (
	DNSChunkSize         = 16
	DNSHostAddressParser = regexp.MustCompile("^([^@]+)@([^:]+):([\\d]+)$")
	DNSAddressParser     = regexp.MustCompile("^([^:]+):([\\d]+)$")
	DNSQuestionParser    = regexp.MustCompile("^([a-fA-F0-9]{42})\\.(.+)\\.$")
)

type DNSClient struct {
	client *dns.Client
	seqn   uint32
}

type DNSServer struct {
	server dns.Server
	mutex  *sync.Mutex
	data   []byte
	cond   *sync.Cond
}

type DNSChannel struct {
	is_client bool
	domain    string
	address   string
	port      int
	server    DNSServer
	client    DNSClient
	stats     Stats
}

func NewDNSChannel() *DNSChannel {
	s := &DNSChannel{
		is_client: true,
		domain:    "",
		address:   "",
		port:      53,

		server: DNSServer{
			server: dns.Server{Addr: ":53", Net: "udp"},
			mutex:  &sync.Mutex{},
			cond:   nil,
			data:   nil,
		},

		client: DNSClient{
			client: nil,
			seqn:   0,
		},
	}

	s.server.cond = sync.NewCond(s.server.mutex)

	return s
}

func (c *DNSChannel) Name() string {
	return "dns"
}

func (c *DNSChannel) Description() string {
	return "As input, read data from incoming DNS requests (example server: dns:192.168.1.2:5353), as output write data as DNS requests (example client: dns:example.com@192.168.1.2:5353)."
}

func (c *DNSChannel) SetData(data []byte) {
	c.server.mutex.Lock()
	defer c.server.mutex.Unlock()
	c.server.data = data
	c.server.cond.Signal()
}

func (c *DNSChannel) GetData() []byte {
	if c.server.data == nil {
		c.server.mutex.Lock()
		defer c.server.mutex.Unlock()
		c.server.cond.Wait()
	}
	return c.server.data
}

func (c *DNSChannel) setupServer(args string) error {
	c.is_client = false

	if c.address != "" {
		c.server.server.Addr = fmt.Sprintf("%s:%d", c.address, c.port)
	}

	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		if len(r.Question) == 1 {
			// fmt.Fprintf(os.Stderr, "Got question: %s\n", r.Question[0].Name)
			if m := DNSQuestionParser.FindStringSubmatch(r.Question[0].Name); len(m) == 3 {
				chunk := m[1]
				domain := m[2]
				// fmt.Fprintf(os.Stderr, "%s", chunk)
				if c.domain == "" || c.domain == domain {
					seqn_hex := chunk[0:8]
					size_hex := chunk[8:10]
					data_hex := chunk[10:]

					// fmt.Fprintf(os.Stderr, "  seqn_hex=%s size_hex=%s data_hex=%s\n", seqn_hex, size_hex, data_hex)

					if seqn_raw, err := hex.DecodeString(seqn_hex); err == nil {
						if size_raw, err := hex.DecodeString(size_hex); err == nil {
							if data_raw, err := hex.DecodeString(data_hex); err == nil {
								// TODO: Check sequence number
								_ = binary.BigEndian.Uint32(seqn_raw)
								size := int(size_raw[0])
								// TODO: Check size vs buffer len

								// fmt.Fprintf(os.Stderr, "Size = %d\n", size)

								final := data_raw[:size]

								// fmt.Fprintf(os.Stderr, "size=%d data='%s'\n", size, string(data_raw))

								c.stats.TotalRead += len(final)
								c.SetData(final)
							}
						}
					}
				}
			}
		}

		m := new(dns.Msg)
		m.SetReply(r)
		w.WriteMsg(m)
	})

	return nil
}

func (c *DNSChannel) setupClient(args string) error {
	c.is_client = true
	if c.address != "" {
		c.client.client = new(dns.Client)
	} else {
		c.client.client = nil
	}

	return nil
}

func (c *DNSChannel) Setup(direction Direction, args string) (err error) {
	if m := DNSHostAddressParser.FindStringSubmatch(args); len(m) > 0 {
		c.domain = m[1]
		c.address = m[2]
		if c.port, err = strconv.Atoi(m[3]); err != nil {
			return err
		}

	} else if m := DNSAddressParser.FindStringSubmatch(args); len(m) > 0 {
		c.address = m[1]
		if c.port, err = strconv.Atoi(m[2]); err != nil {
			return err
		}
	}

	if direction == INPUT_CHANNEL {
		return c.setupServer(args)
	} else {
		return c.setupClient(args)
	}

	return nil
}

func (c *DNSChannel) Start() error {
	if c.is_client == true {
		fmt.Fprintf(os.Stderr, "Performing DNS lookups ...\n")
	} else {
		fmt.Fprintf(os.Stderr, "Running DNS server on '%s:%d' ...\n", c.address, c.port)

		go func() {
			if err := c.server.server.ListenAndServe(); err != nil {
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

	data := c.GetData()
	if data == nil {
		return 0, fmt.Errorf("EOF")
	} else if len(b) < len(data) {
		return 0, fmt.Errorf("Need more space.")
	}

	c.SetData(nil)

	for i, c := range data {
		b[i] = c
	}

	return len(data), nil
}

func (c *DNSChannel) Lookup(fqdn string) error {
	// fmt.Fprintf(os.Stderr, "Resolving %s (seqn=0x%x) ...\n", fqdn, c.client.seqn)

	if c.client.client == nil {
		if _, err := net.LookupHost(fqdn); err != nil {
			return err
		}

	} else {
		m1 := new(dns.Msg)
		m1.Id = dns.Id()
		m1.RecursionDesired = true
		m1.Question = make([]dns.Question, 1)
		m1.Question[0] = dns.Question{fqdn + ".", dns.TypeA, dns.ClassINET}

		if _, _, err := c.client.client.Exchange(m1, fmt.Sprintf("%s:%d", c.address, c.port)); err != nil {
			return err
		}
	}

	return nil
}

func (c *DNSChannel) Write(b []byte) (n int, err error) {
	if c.is_client == false {
		return 0, fmt.Errorf("dns server can't be used for writing.")
	}

	// fmt.Printf("Writing %d bytes: '%s' -> %s...\n", len(b), string(b), hex.EncodeToString(b))

	total_size := len(b)
	left := total_size
	done := 0

	for left > 0 {
		size := DNSChunkSize
		if left < size {
			size = left
		}

		// fmt.Printf("  chunk := b[%d:%d]\n", done, size)
		chunk := b[done : done+size]
		// add padding
		if size < DNSChunkSize {
			// fmt.Printf("    padding\n")
			pad := size
			for pad < DNSChunkSize {
				chunk = append(chunk, 0x00)
				pad++
			}
		}

		seqn_buffer := make([]byte, 4)
		binary.BigEndian.PutUint32(seqn_buffer, c.client.seqn)

		size_buffer := make([]byte, 1)
		size_buffer[0] = byte(size & 0xff)
		// fmt.Fprintf(os.Stderr, "size=%d\n", size)

		chunk = append(size_buffer, chunk...)
		chunk = append(seqn_buffer, chunk...)

		fqdn := fmt.Sprintf("%s.%s", hex.EncodeToString(chunk), c.domain)

		if err := c.Lookup(fqdn); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		} else {
			c.stats.TotalWrote += size
		}

		done += size
		left -= size
		c.client.seqn++
	}

	return done, nil
}

func (c *DNSChannel) Stats() Stats {
	return c.stats
}
