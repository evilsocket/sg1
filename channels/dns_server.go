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
	"os"
	"strings"
	"sync"
)

type DNSServer struct {
	server dns.Server

	mutex *sync.Mutex
	data  []byte
	cond  *sync.Cond
	stats Stats
}

func NewDNSServerChannel() *DNSServer {
	s := &DNSServer{
		server: dns.Server{Addr: ":53", Net: "udp"},
		mutex:  &sync.Mutex{},
		cond:   nil,
		data:   nil,
	}

	s.cond = sync.NewCond(s.mutex)

	return s
}

func (c *DNSServer) Name() string {
	return "dnsserver"
}

func (c *DNSServer) Description() string {
	return "Read data incoming as DNS requests (example: dnsserver:192.168.1.2:5353)."
}

func (c *DNSServer) SetData(data []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = data
	c.cond.Signal()
}

func (c *DNSServer) GetData() []byte {
	if c.data == nil {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.cond.Wait()
	}
	return c.data
}

func (c *DNSServer) Setup(direction Direction, args string) error {
	if args != "" {
		c.server.Addr = args
	}

	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		// TODO: This is horrible, refactor with proper packet parsing.
		if len(r.Question) == 1 {
			parts := strings.SplitN(r.Question[0].Name, ".", 2)
			if len(parts) == 2 {
				chunk := parts[0]
				if len(chunk) == 40 {
					seqn_hex := chunk[0:8]
					data_hex := chunk[8:]

					if seqn_raw, err := hex.DecodeString(seqn_hex); err == nil {
						if data_raw, err := hex.DecodeString(data_hex); err == nil {
							// TODO: Check sequence number
							_ = binary.BigEndian.Uint32(seqn_raw)

							c.stats.TotalRead += len(data_raw)
							c.SetData(data_raw)
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

func (c *DNSServer) Start() error {
	fmt.Fprintf(os.Stderr, "Running DNS server on '%s' ...\n", c.server.Addr)

	go func() {
		if err := c.server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	return nil
}

func (c *DNSServer) HasReader() bool {
	return true
}

func (c *DNSServer) Read(b []byte) (n int, err error) {
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

	return len(b), nil
}

func (c *DNSServer) HasWriter() bool {
	return false
}

func (c *DNSServer) Write(b []byte) (n int, err error) {
	return 0, fmt.Errorf("dnsserver can't be used for writing.")
}

func (c *DNSServer) Stats() Stats {
	return c.stats
}
