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
	"github.com/evilsocket/sg1"
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
	return "Read from input, encrypt or decrypt in AES and write to output ( use -aes-key and -aes-mode arguments )."
}

func (m *AES) Register() error {
	flag.StringVar(&m.key, "aes-key", "", "AES key.")
	flag.StringVar(&m.mode, "aes-mode", "encrypt", "AES mode, can be 'encrypt' or 'decrypt'.")
	return nil
}

func (m *AES) getCipher(keystring string) (cipher.Block, error) {
	key := []byte(keystring)
	ksize := len(key)

	for _, min_sz := range []int{16, 24, 32} {
		if ksize < min_sz {
			sg1.Warning("AES key size %d is less than %d, key will be padded with 0s.\n", ksize, min_sz)
			ksize = min_sz
			key = channels.PadBuffer(key, ksize, 0x00)
			break
		} else if ksize > 32 {
			sg1.Warning("AES key size %d is greater than 32, key will be shortened.\n", ksize)
			ksize = 32
			key = key[0:ksize]
			break
		}

		if ksize == min_sz {
			break
		}
	}

	return aes.NewCipher(key)
}

func (m *AES) encrypt(plaintext []byte, keystring string) (error, []byte) {
	block, err := m.getCipher(keystring)
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
	block, err := m.getCipher(keystring)
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

	return ReadLoop(input, output, buffer_size, delay, func(buff []byte) (int, []byte, error) {
		var err error
		var data []byte

		if m.mode == "encrypt" {
			size := len(buff)
			sg1.Debug("AES encrypting %d bytes ...\n", size)
			packet := channels.NewPacket(0, uint32(size), buff)
			buff = packet.Raw()
			sg1.Debug("Packet: %s\n", sg1.Hex(buff))

			err, data = m.encrypt(buff, m.key)
		} else if m.mode == "decrypt" {
			sg1.Debug("AES decrypting %d bytes ...\n", len(buff))
			err, data = m.decrypt(buff, m.key)
			if err == nil {
				if packet, err := channels.DecodePacket(data); err == nil {
					sg1.Debug("AES decrypted packet of %d bytes.\n", packet.DataSize)
					sg1.Debug("Packet data: %s\n", sg1.Hex(packet.Data))
					data = packet.Data
				}
			}
		} else {
			err = fmt.Errorf("Unhandled AES mode '%s'.", m.mode)
		}

		return len(data), data, err
	})
}
