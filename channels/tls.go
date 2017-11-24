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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"github.com/evilsocket/sg1"
	"math/big"
	"net"
	"os"
	"sync"
	"time"
)

type TLSChannel struct {
	pem_file  string
	key_file  string
	address   string
	is_client bool
	config    *tls.Config

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
		config:     nil,
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
	return "Read or write data on a TLS server (for input) or client (for output) connection, if no --tls-key and --tls-pem parameters are passed it will take care of certificate generation ( example: tls:127.0.0.1:8083 )."
}

func (c *TLSChannel) Register() error {
	flag.StringVar(&c.pem_file, "tls-pem", "", "PEM file for TLS connection.")
	flag.StringVar(&c.key_file, "tls-key", "", "KEY file for TLS connection.")
	return nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey

	case *ecdsa.PrivateKey:
		return &k.PublicKey

	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}

	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			sg1.Log("Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}

		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}

	default:
		return nil
	}
}

func (c *TLSChannel) getCertificateConfig() (conf *tls.Config, err error) {
	if c.pem_file == "" || c.key_file == "" {
		sg1.Log("Generating new ECDSA certificate ...\n\n")

		priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		if err != nil {
			return conf, err
		}

		notBefore := time.Now()
		notAfter := notBefore.Add(time.Duration(24) * time.Hour)
		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
		if err != nil {
			return conf, err
		}

		template := x509.Certificate{
			SerialNumber:          serialNumber,
			Subject:               pkix.Name{Organization: []string{"SG1 Co"}},
			NotBefore:             notBefore,
			NotAfter:              notAfter,
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
		}

		cert_raw, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
		if err != nil {
			return conf, err
		}

		pem_cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert_raw})
		pem_key := pem.EncodeToMemory(pemBlockForKey(priv))

		cert, err := tls.X509KeyPair(pem_cert, pem_key)
		if err != nil {
			return conf, err
		}

		return &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}, nil
	} else {
		sg1.Log("Loading X509 key pair from %s (%s).\n\n", c.pem_file, c.key_file)

		cert, err := tls.LoadX509KeyPair(c.pem_file, c.key_file)
		if err != nil {
			return conf, err
		}

		return &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}, nil
	}
}

func (c *TLSChannel) Setup(direction Direction, args string) (err error) {
	c.config, err = c.getCertificateConfig()
	if err != nil {
		return err
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
		if c.connection, err = tls.Dial("tcp", c.address, c.config); err != nil {
			return err
		}
	} else {
		if c.listener, err = tls.Listen("tcp", c.address, c.config); err != nil {
			return err
		}

		go func() {
			for {
				if conn, err := c.listener.Accept(); err == nil {
					c.SetClient(conn)
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

func (c *TLSChannel) SetClient(con net.Conn) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		c.client.Close()
		c.client = nil
	}

	c.client = con
	c.cond.Signal()
}

func (c *TLSChannel) WaitForClient() {
	if c.is_client == false && c.client == nil {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.cond.Wait()
	}
}

func (c *TLSChannel) GetClient() net.Conn {
	c.WaitForClient()
	return c.client
}

func (c *TLSChannel) Read(b []byte) (n int, err error) {
	if c.is_client == false {
		n, err = c.GetClient().Read(b)
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
		n, err = c.GetClient().Write(b)
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
