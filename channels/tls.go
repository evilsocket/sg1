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
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"sync"
)

type TLSChannel struct {
	pem_file  string
	key_file  string
	address   string
	is_client bool
	config    tls.Config

	connection *tls.Conn
	listener   net.Listener
	client     net.Conn

	mutex *sync.Mutex
	cond  *sync.Cond
	stats Stats
}

func NewTLSChannel() *TLSChannel {
	s := &TLSChannel{
		pem_file:   "",
		key_file:   "",
		address:    "",
		is_client:  true,
		connection: nil,
		client:     nil,
		listener:   nil,
		mutex:      &sync.Mutex{},
		cond:       nil,
	}

	s.cond = sync.NewCond(s.mutex)
	return s
}

func (c *TLSChannel) Name() string {
	return "tls"
}

func (c *TLSChannel) Description() string {
	return "Read or write data on a TLS server (for input) or client (for output) connection, requires --tls-key and --tls-pem parameters ( example: tls:127.0.0.1:8083 )."
}

func (c *TLSChannel) Register() error {
	flag.StringVar(&c.pem_file, "tls-pem", "", "PEM file for TLS connection.")
	flag.StringVar(&c.key_file, "tls-key", "", "KEY file for TLS connection.")
	return nil
}

func (c *TLSChannel) Setup(direction Direction, args string) (err error) {
	if c.pem_file == "" {
		return fmt.Errorf("No --tls-pem file specified.")
	} else if c.key_file == "" {
		return fmt.Errorf("No --tls-key file specified.")
	}

	cert, err := tls.LoadX509KeyPair(c.pem_file, c.key_file)
	if err != nil {
		return err
	}

	c.config = tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	if direction == INPUT_CHANNEL {
		c.is_client = false

	} else {
		c.is_client = true
	}

	c.address = args

	return nil
}

func (c *TLSChannel) Start() (err error) {
	if c.is_client {
		if c.connection, err = tls.Dial("tcp", c.address, &c.config); err != nil {
			return err
		}
	} else {
		if c.listener, err = tls.Listen("tcp", c.address, &c.config); err != nil {
			return err
		}

		go func() {
			for {
				if conn, err := c.listener.Accept(); err == nil {
					c.client = conn
					c.mutex.Lock()
					c.cond.Signal()
					c.mutex.Unlock()
				} else {
					break
				}
			}
		}()
	}

	return nil
}

func (c *TLSChannel) HasReader() bool {
	return true
}

func (c *TLSChannel) HasWriter() bool {
	return true
}

func (c *TLSChannel) WaitForClient() {
	if c.is_client == false && c.client == nil {
		c.mutex.Lock()
		c.cond.Wait()
		c.mutex.Unlock()
	}
}

func (c *TLSChannel) Read(b []byte) (n int, err error) {
	if c.is_client == false {
		c.WaitForClient()
		n, err = c.client.Read(b)
	} else {
		n, err = c.connection.Read(b)
	}

	if n > 0 {
		c.stats.TotalRead += n
	}
	return n, err
}

func (c *TLSChannel) Write(b []byte) (n int, err error) {
	if c.is_client == false {
		c.WaitForClient()
		n, err = c.client.Write(b)
	} else {
		n, err = c.connection.Write(b)
	}

	if n > 0 {
		c.stats.TotalWrote += n
	}
	return n, err
}

func (c *TLSChannel) Stats() Stats {
	return c.stats
}
