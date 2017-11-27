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
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/evilsocket/sg1/sg1"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	DefaultStreamName = "PBSTREAM"
)

var argsParser = regexp.MustCompile("^([a-fA-F0-9]{32})/([a-fA-F0-9]{32})(#.+)?$")

type Pastebin struct {
	api       *PastebinAPI
	is_client bool
	preserve  bool
	stream    string
	seq       *sg1.PacketSequencer
	poll_time int
	stats     Stats
}

func NewPastebinChannel() *Pastebin {
	return &Pastebin{
		api:       nil,
		is_client: true,
		stream:    DefaultStreamName,
		preserve:  false,
		poll_time: 1000,
		seq:       sg1.NewPacketSequencer(),
	}
}

func (c *Pastebin) Copy() interface{} {
	return NewPastebinChannel()
}

func (c *Pastebin) Name() string {
	return "pastebin"
}

func (c *Pastebin) Register() error {
	flag.BoolVar(&c.preserve, "pastebin-preserve", c.preserve, "Do not delete pastes after reading them.")
	flag.IntVar(&c.poll_time, "pastebin-poll-time", c.poll_time, "Number of milliseconds to wait between one pastebin API request and another.")
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

	if m := argsParser.FindStringSubmatch(args); len(m) == 4 {
		c.api = NewPastebinAPI(m[1], m[2])
		if len(m[3]) > 1 {
			c.stream = m[3][1:]
		}
	} else {
		return fmt.Errorf("Usage: pastebin:YOUR-API-DEV-KEY/YOUR-API-USER-KEY(#stream_name)?")
	}

	sg1.Debug("Setup pastebin channel: direction=%d api_key='%s' user_key='%s' stream='%s'\n", direction, c.api.ApiKey, c.api.UserKey, c.stream)

	return nil
}

func (c *Pastebin) Start() error {
	c.seq.Start()

	if c.is_client == true {
		sg1.Log("Sending data to pastebin ...\n")
	} else {
		sg1.Log("Running pastebin listener ...\n\n")

		go func() {
			var err error
			var pastes = make([]XmlPaste, 0)
			var wait = true

			for {
				// if we don't have pastes in the queue to process
				if pastes == nil || len(pastes) == 0 {
					sg1.Debug("No queued pastes, requesting to API ...\n")

					// request new pastes
					pastes, err = c.api.GetPastes()
					if err != nil {
						sg1.Error("Error while requesting pastes: %s.\n", err)
						continue
					}

					sg1.Debug("Filtering %d pastes by stream '%s'.\n", len(pastes), c.stream)

					// filter by stream name
					filtered := make([]XmlPaste, 0)
					for _, paste := range pastes {
						if strings.Contains(paste.title, c.stream) == true {
							filtered = append(filtered, paste)
						}
					}

					pastes = filtered

					sg1.Debug("Filtered pastes are now %d.\n", len(pastes))
				} else {
					sg1.Debug("Got %d pastes to process.\n", len(pastes))
				}

				n_available := len(pastes)
				if n_available > 0 {
					// sort by title (which is the hex encoded timestamp) and get oldest
					sort.Slice(pastes, func(i, j int) bool {
						return pastes[i].title > pastes[j].title
					})
					oldest := pastes[0]
					pastes = pastes[1:]
					// if we have more than one paste available, this will make the loop
					// skip the sleep, pass the first if without requesting new pastes
					// and keep getting the ordered paste until the size of the queue
					// is 1 again.
					wait = (n_available == 1)

					sg1.Debug("Oldest paste to process is %s, requesting to API ...\n", oldest.key)

					paste, err := c.api.GetPaste(oldest.key)
					if err != nil {
						sg1.Error("Error while requesting paste %s: %s\n", oldest.key, err)
						continue
					}

					sg1.Debug("Decoding paste body of %d bytes.\n", len(paste))
					chunk, err := hex.DecodeString(paste)
					if err != nil {
						sg1.Error("Error while decoding body from hex '%s': %s\n", paste, err)
						continue
					}

					sg1.Debug("Decoding packet from %d bytes.\n", len(chunk))
					if packet, err := sg1.DecodePacket(chunk); err == nil {
						sg1.Debug("Decoded packet of %d bytes.\n", packet.DataSize)

						c.stats.TotalRead += int(packet.DataSize)
						c.seq.Add(packet)
					} else {
						sg1.Error("Error while decoding body: %s\n", err)
					}

					if c.preserve == false {
						sg1.Debug("Deleting paste %s.\n", oldest.key)
						_, err = c.api.DeletePaste(oldest)
						if err != nil {
							sg1.Error("Error while deleting paste %s: %s\n", oldest.key, err)
						}
					}
				}

				if wait {
					sg1.Debug("Waiting for %d milliseconds ...\n", c.poll_time)
					time.Sleep(time.Duration(c.poll_time) * time.Millisecond)
				}
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
	packet := c.seq.Get()
	data := packet.Data
	size := len(data)
	for i, c := range data {
		b[i] = c
	}

	sg1.Debug("Read %d bytes from pastebin channel.\n", size)

	return size, nil
}

func (c *Pastebin) Write(b []byte) (n int, err error) {
	packet := c.seq.Packet(b, 1)
	size := len(b)
	paste := Paste{
		Text:       packet.Hex(),
		Name:       fmt.Sprintf("SG1 %s 0x%x", c.stream, sg1.Time()),
		Privacy:    Private,
		ExpireDate: Hour,
	}

	sg1.Log("Sending paste for payload of %d bytes, paste text is %d bytes.\n", len(b), len(paste.Text))

	resp, err := c.api.CreatePaste(paste)
	if err != nil {
		return 0, err
	} else if strings.Contains(resp, "://") {
		sg1.Raw("\n%s\n", resp)

		c.stats.TotalWrote += size
		sg1.Debug("Wrote %d bytes to pastebin channel.\n", size)

		return n, nil
	} else {
		return 0, fmt.Errorf("Could not send paste: %s", resp)
	}
}

func (c *Pastebin) Stats() Stats {
	return c.stats
}
