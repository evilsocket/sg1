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
	"github.com/evilsocket/sg1"
	"net"
	"sync"
)

type TCPChannel struct {
	is_client  bool
	address    *net.TCPAddr
	connection *net.TCPConn
	client     net.Conn
	listener   net.Listener
	mutex      *sync.Mutex
	cond       *sync.Cond
	stats      Stats
}

func NewTCPChannel() *TCPChannel {
	s := &TCPChannel{
		is_client:  true,
		address:    nil,
		connection: nil,
		mutex:      &sync.Mutex{},
		cond:       nil,
	}

	s.cond = sync.NewCond(s.mutex)
	return s
}

func (c *TCPChannel) Copy() interface{} {
	return NewTCPChannel()
}

func (c *TCPChannel) Name() string {
	return "tcp"
}

func (c *TCPChannel) Description() string {
	return "Read or write data on a TCP server (for input) or client (for output) connection ( example: tcp:127.0.0.1:8080 )."
}

func (c *TCPChannel) Register() error {
	return nil
}

func (c *TCPChannel) Setup(direction Direction, args string) (err error) {
	if direction == INPUT_CHANNEL {
		c.is_client = false

	} else {
		c.is_client = true
	}

	if c.address, err = net.ResolveTCPAddr("tcp", args); err != nil {
		return err
	}

	sg1.Debug("Setup tcp channel: direction=%d address=%s\n", direction, args)

	return nil
}

func (c *TCPChannel) Start() (err error) {
	if c.is_client {
		sg1.Log("Connecting to TCP endpoint %s ...\n\n", c.address)

		if c.connection, err = net.DialTCP("tcp", nil, c.address); err != nil {
			return err
		}
	} else {
		if c.listener, err = net.ListenTCP("tcp", c.address); err != nil {
			return err
		}

		go func() {
			sg1.Log("Started TCP server on %s ...\n\n", c.address)

			for {
				if conn, err := c.listener.Accept(); err == nil {
					sg1.Debug("Got client connection %v.\n", conn)
					c.SetClient(conn)
				} else {
					sg1.Error("Error while accepting connection: %s\n", err)
					break
				}
			}
		}()
	}

	return nil
}

func (c *TCPChannel) HasReader() bool {
	return true
}

func (c *TCPChannel) HasWriter() bool {
	return true
}

func (c *TCPChannel) SetClient(con net.Conn) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	sg1.Debug("Setting client.\n")

	if c.client != nil {
		c.client.Close()
		c.client = nil
	}

	c.client = con
	c.cond.Signal()
}

func (c *TCPChannel) WaitForClient() {
	if c.is_client == false && c.client == nil {
		sg1.Debug("Waiting for client ...\n")
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.cond.Wait()
		sg1.Debug("Got client.\n")
	}
}

func (c *TCPChannel) GetClient() net.Conn {
	c.WaitForClient()
	return c.client
}

func (c *TCPChannel) Read(b []byte) (n int, err error) {
	if c.is_client == false {
		n, err = c.GetClient().Read(b)

		sg1.Debug("Read %d bytes from client.\n", n)
	} else {
		n, err = c.connection.Read(b)

		sg1.Debug("Read %d bytes from server.\n", n)
	}

	if n > 0 {
		c.stats.TotalRead += n
	}
	return n, err
}

func (c *TCPChannel) Write(b []byte) (n int, err error) {
	if c.is_client == false {
		n, err = c.GetClient().Write(b)

		sg1.Debug("Wrote %d bytes to client.\n", n)
	} else {
		n, err = c.connection.Write(b)

		sg1.Debug("Wrote %d bytes to server.\n", n)
	}

	if n > 0 {
		c.stats.TotalWrote += n
	}
	return n, err
}

func (c *TCPChannel) Stats() Stats {
	return c.stats
}
