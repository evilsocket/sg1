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
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	Never       = "N"
	TenMinutues = "10M"
	Hour        = "1H"
	Day         = "1D"
	Week        = "1W"
	TwoWeeks    = "2W"
	Month       = "1M"

	Public   = "0"
	Unlisted = "1"
	Private  = "2"
)

var argsParser = regexp.MustCompile("^([a-fA-F0-9]{32})/([a-fA-F0-9]{32})$")

type XmlPaste struct {
	key   string
	title string
}

type Paste struct {
	Text       string
	Name       string
	Privacy    string
	ExpireDate string
}

type Pastebin struct {
	is_client bool
	api_key   string
	user_key  string
	seqn      uint32
	chunks    chan []byte
	stats     Stats
}

func NewPastebinChannel() *Pastebin {
	return &Pastebin{
		is_client: true,
		chunks:    make(chan []byte),
	}
}

func (c *Pastebin) Name() string {
	return "pastebin"
}

func (c *Pastebin) Register() error {
	flag.StringVar(&c.api_key, "pastebin-api-key", "", "API developer key for the pastebin channel.")
	flag.StringVar(&c.user_key, "pastebin-user-key", "", "API user key for the pastebin channel ( https://pastebin.com/api#8 ).")
	return nil
}

func (c *Pastebin) Description() string {
	return "Read data from pastebin of a given user and write data as pastebins to that user account."
}

func (c *Pastebin) Setup(direction Direction, args string) error {
	if direction == INPUT_CHANNEL {
		c.is_client = false
	} else {
		c.is_client = true
	}

	if m := argsParser.FindStringSubmatch(args); len(m) == 3 {
		c.api_key = m[1]
		c.user_key = m[2]
	} else {
		return fmt.Errorf("Usage: pastebin:YOUR-API-DEV-KEY/YOUR-API-USER-KEY")
	}

	return nil
}

func (c *Pastebin) Start() error {
	if c.is_client == true {
		fmt.Fprintf(os.Stderr, "Sending data to pastebin ...\n")
	} else {
		fmt.Fprintf(os.Stderr, "Running pastebin listener ...\n\n")

		go func() {
			for {
				pastes, err := c.getPastes()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error while requesting pastes: %s.\n", err)
					continue
				}

				if len(pastes) > 0 {
					// sort by title, get oldest and delete it
					sort.Slice(pastes, func(i, j int) bool {
						return pastes[i].title > pastes[j].title
					})

					oldest := pastes[0]

					// fmt.Fprintf(os.Stderr, "Requesting paste %v\n", oldest)

					paste, err := c.getPaste(oldest.key)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error while requesting paste %s: %s\n", oldest.key, err)
					}

					// fmt.Fprintf(os.Stderr, "Decoding paste body:\n%s\n", paste)
					chunk, err := hex.DecodeString(paste)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error while decoding body from hex: %s\n", err)
					}

					// fmt.Fprintf(os.Stderr, "Decoding packet from %d bytes.\n", len(chunk))
					if packet, err := DecodePacket(chunk); err == nil {
						// fix data size
						packet.DataSize = uint32(len(packet.Data))

						// fmt.Fprintf(os.Stderr, "  packet.DataSize = %d\n", packet.DataSize)
						// fmt.Fprintf(os.Stderr, "  packet.Data is %s\n", hex.EncodeToString(packet.Data))
						// fmt.Fprintf(os.Stderr, "  packet.Data is %d bytes\n", len(packet.Data))

						c.stats.TotalRead += int(packet.DataSize)
						c.chunks <- packet.Data
					} else {
						fmt.Fprintf(os.Stderr, "Error while decoding body: %s\n", err)
					}

					// fmt.Fprintf(os.Stderr, "Deleting paste %s.\n", oldest.key)
					_, err = c.deletePaste(oldest)
				}

				time.Sleep(time.Duration(1) * time.Second)
			}
		}()
	}

	return nil
}

func (c *Pastebin) HasReader() bool {
	return true
}

func (c *Pastebin) HasWriter() bool {
	return true
}

func (c *Pastebin) Read(b []byte) (n int, err error) {
	data := <-c.chunks
	for i, c := range data {
		b[i] = c
	}
	size := len(data)
	c.stats.TotalRead += size
	return size, nil
}

func (c *Pastebin) Write(b []byte) (n int, err error) {
	packet := NewPacket(c.seqn, uint32(len(b)), b)

	paste := Paste{
		Text:       hex.EncodeToString(packet.Encode()),
		Name:       fmt.Sprintf("SG1 0x%x", time.Now().UnixNano()/int64(time.Millisecond)),
		Privacy:    Private,
		ExpireDate: Hour,
	}

	fmt.Fprintf(os.Stderr, "Sending paste for payload of %d bytes, paste text is %d bytes.\n", len(b), len(paste.Text))

	resp, err := c.sendPaste(paste)

	if err != nil {
		return 0, err
	} else if strings.Contains(resp, "://") {
		fmt.Fprintf(os.Stderr, "\n%s\n", resp)
		c.seqn++
		c.stats.TotalWrote += len(b)
		return n, nil
	} else {
		return 0, fmt.Errorf("Could not send paste: %s", resp)
	}
}

func (c *Pastebin) Stats() Stats {
	return c.stats
}

func (c *Pastebin) apiRequest(page string, values url.Values) (body string, err error) {
	response, err := http.PostForm("https://pastebin.com/api/"+page, values)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", fmt.Errorf("Got response code %d.", response.StatusCode)
	}

	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(response.Body)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (c *Pastebin) getPaste(key string) (body string, err error) {
	values := url.Values{}
	values.Set("api_dev_key", c.api_key)
	values.Set("api_user_key", c.user_key)
	values.Set("api_paste_key", key)
	values.Set("api_option", "show_paste")

	return c.apiRequest("api_raw.php", values)
}

func (c *Pastebin) parseXmlPastes(body string) []XmlPaste {
	keyParser := regexp.MustCompile("^<paste_key>(.+)</paste_key>$")
	titleParser := regexp.MustCompile("^<paste_title>SG1 (.+)</paste_title>$")
	pastes := make([]XmlPaste, 0)
	lines := strings.Split(body, "\n")
	paste := XmlPaste{}

	for _, line := range lines {
		line = strings.Trim(line, " \n\r\t")
		// skip empty lines
		if line == "" {
			continue
		} else if line == "<paste>" {
			paste = XmlPaste{}
		} else if line == "</paste>" {
			pastes = append(pastes, paste)
		} else if m := keyParser.FindStringSubmatch(line); len(m) == 2 {
			paste.key = m[1]
		} else if m := titleParser.FindStringSubmatch(line); len(m) == 2 {
			paste.title = m[1]
		}
	}

	return pastes
}

func (c *Pastebin) getPastes() (pastes []XmlPaste, err error) {
	values := url.Values{}
	values.Set("api_dev_key", c.api_key)
	values.Set("api_user_key", c.user_key)
	values.Set("api_option", "list")
	values.Set("api_results_limit", "1000")

	body, err := c.apiRequest("api_post.php", values)
	if err != nil {
		return nil, err
	}

	return c.parseXmlPastes(body), nil
}

func (c *Pastebin) deletePaste(paste XmlPaste) (resp string, err error) {
	values := url.Values{}
	values.Set("api_dev_key", c.api_key)
	values.Set("api_user_key", c.user_key)
	values.Set("api_paste_key", paste.key)
	values.Set("api_option", "delete")

	return c.apiRequest("api_post.php", values)
}

func (c *Pastebin) sendPaste(paste Paste) (resp string, err error) {
	values := url.Values{}
	values.Set("api_dev_key", c.api_key)
	values.Set("api_user_key", c.user_key)
	values.Set("api_option", "paste")
	values.Set("api_paste_code", paste.Text)
	values.Set("api_paste_name", paste.Name)
	values.Set("api_paste_private", paste.Privacy)
	values.Set("api_paste_expire_date", paste.ExpireDate)

	return c.apiRequest("api_post.php", values)
}
