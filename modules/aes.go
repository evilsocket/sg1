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
	"time"
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
	return "Read from input, encrypt or decrypt in AES and write to output ( use -aes-key and -aes-mode arguments )."
}

func (m *AES) Register() error {
	flag.StringVar(&m.key, "aes-key", "", "AES key.")
	flag.StringVar(&m.mode, "aes-mode", "encrypt", "AES mode, can be 'encrypt' or 'decrypt'.")
	return nil
}

func (m *AES) encrypt(plaintext []byte, keystring string) (error, []byte) {
	key := []byte(keystring)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err, nil
	}

	// Empty array of 16 + plaintext length
	// Include the IV at the beginning
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	// Slice of first 16 bytes
	iv := ciphertext[:aes.BlockSize]
	// Write 16 rand bytes to fill iv
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err, nil
	}

	// Return an encrypted stream
	stream := cipher.NewCFBEncrypter(block, iv)
	// Encrypt bytes from plaintext to ciphertext
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return nil, ciphertext
}

func (m *AES) decrypt(ciphertext []byte, keystring string) (error, []byte) {
	key := []byte(keystring)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err, nil
	}

	// Before even testing the decryption,
	// if the text is too small, then it is incorrect
	if len(ciphertext) < aes.BlockSize {
		return fmt.Errorf("Text is too short"), nil
	}

	// Get the 16 byte IV
	iv := ciphertext[:aes.BlockSize]
	// Remove the IV from the ciphertext
	ciphertext = ciphertext[aes.BlockSize:]
	// Return a decrypted stream
	stream := cipher.NewCFBDecrypter(block, iv)
	// Decrypt bytes from ciphertext
	stream.XORKeyStream(ciphertext, ciphertext)

	return nil, ciphertext
}

func (m *AES) Run(input, output channels.Channel, buffer_size, delay int) error {
	if m.key == "" {
		return fmt.Errorf("No AES key specified.")
	}

	var n int
	var err error
	var buff = make([]byte, buffer_size)

	for {
		if n, err = input.Read(buff); err != nil {
			if err.Error() == "EOF" {
				break
			} else {
				return err
			}
		}

		var err error
		var data []byte

		if m.mode == "encrypt" {
			err, data = m.encrypt(buff[:n], m.key)
		} else if m.mode == "decrypt" {
			err, data = m.decrypt(buff[:n], m.key)
		}

		if err != nil {
			return err
		}

		if _, err = output.Write(data); err != nil {
			return err
		}

		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}

	return nil
}
