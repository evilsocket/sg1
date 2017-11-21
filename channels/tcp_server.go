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
	"net"
	"sync"
)

type TCPServer struct {
	address    *net.TCPAddr
	listener   net.Listener
	mutex      *sync.Mutex
	cond       *sync.Cond
	connection net.Conn
}

func NewTCPServerChannel() *TCPServer {
	s := &TCPServer{
		address: nil,
		mutex:   &sync.Mutex{},
		cond:    nil,
	}

	s.cond = sync.NewCond(s.mutex)
	return s
}

func (c *TCPServer) Name() string {
	return "tcpserver"
}

func (c *TCPServer) Description() string {
	return "Read or write data on a TCP server connection ( example: tcpserver:127.0.0.1:8080 )."
}

func (c *TCPServer) SetArgs(args string) (err error) {
	if c.address, err = net.ResolveTCPAddr("tcp", args); err != nil {
		return err
	}

	return nil
}

func (c *TCPServer) Start() (err error) {
	if c.listener, err = net.ListenTCP("tcp", c.address); err != nil {
		return err
	}

	go func() {
		for {
			if conn, err := c.listener.Accept(); err == nil {
				c.connection = conn
				c.mutex.Lock()
				c.cond.Signal()
				c.mutex.Unlock()
			} else {
				break
			}
		}
	}()

	return nil
}

func (c *TCPServer) HasReader() bool {
	return true
}

func (c *TCPServer) HasWriter() bool {
	return true
}

func (c *TCPServer) WaitForClient() {
	if c.connection == nil {
		c.mutex.Lock()
		c.cond.Wait()
		c.mutex.Unlock()
	}
}

func (c *TCPServer) Read(b []byte) (n int, err error) {
	c.WaitForClient()
	return c.connection.Read(b)
}

func (c *TCPServer) Write(b []byte) (n int, err error) {
	c.WaitForClient()
	return c.connection.Write(b)
}
