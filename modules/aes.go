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
package modules

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"flag"
	"fmt"
	"github.com/evilsocket/sg1/channels"
	"io"
)

type AES struct {
	key  string
	mode string
	in   channels.Channel
	out  channels.Channel
}

func NewAES() *AES {
	return &AES{
		key:  "",
		mode: "encrypt",
		in:   nil,
		out:  nil,
	}
}

func (m *AES) Name() string {
	return "aes"
}

func (m *AES) Description() string {
	return "Read from input encrypt or decrypt in AES and write to output ( use -aes-key and -aes-mode arguments )."
}

func (m *AES) Register() error {
	flag.StringVar(&m.key, "aes-key", "", "AES key.")
	flag.StringVar(&m.mode, "aes-mode", "encrypt", "AES mode, can be 'encrypt' or 'decrypt'.")
	return nil
}

func (m *AES) encrypt(block cipher.Block, input, output channels.Channel) error {
	// generate and send the IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	if _, err := output.Write(iv); err != nil {
		return err
	}

	// Return an encrypted stream
	var err error
	var n int
	var buff = make([]byte, aes.BlockSize)
	var ciphertext = make([]byte, aes.BlockSize)

	stream := cipher.NewCFBEncrypter(block, iv)

	for {
		if n, err = input.Read(buff); err != nil {
			if err.Error() == "EOF" {
				break
			} else {
				return err
			}
		}

		stream.XORKeyStream(ciphertext, buff[:n])

		if _, err = output.Write(ciphertext); err != nil {
			return err
		}
	}

	return nil
}

func (m *AES) decrypt(block cipher.Block, input, output channels.Channel) error {
	// read iv
	iv := make([]byte, aes.BlockSize)
	if _, err := input.Read(iv); err != nil {
		return fmt.Errorf("Could not read IV from input channel '%s'.", input.Name())
	}

	var err error
	var ciphertext = make([]byte, aes.BlockSize)
	var n int

	stream := cipher.NewCFBDecrypter(block, iv)

	for {
		// read encrypted blocks
		if n, err = input.Read(ciphertext); err != nil {
			if err.Error() == "EOF" {
				break
			} else {
				return err
			}
		}

		var buff = make([]byte, n)
		stream.XORKeyStream(buff, ciphertext)

		if _, err = output.Write(buff); err != nil {
			return err
		}
	}

	return nil
}

func (m *AES) Run(input, output channels.Channel) error {
	if m.key == "" {
		return fmt.Errorf("No AES key specified.")
	}

	// Create the AES cipher
	block_cipher, err := aes.NewCipher([]byte(m.key))
	if err != nil {
		return err
	}

	if m.mode == "encrypt" {
		return m.encrypt(block_cipher, input, output)
	} else if m.mode == "decrypt" {
		return m.decrypt(block_cipher, input, output)
	} else {
		return fmt.Errorf("Invalid --aes-mode parameter specified.")
	}
}
