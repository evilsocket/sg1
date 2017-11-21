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
	"fmt"
	"strings"
	"sync"
)

var (
	registered = make(map[string]Channel)
	mt         = &sync.Mutex{}
)

func Register(channel Channel) error {
	mt.Lock()
	defer mt.Unlock()

	channel_name := channel.Name()
	if _, found := registered[channel_name]; found {
		return fmt.Errorf("channel with name %s already registered.", channel_name)
	}

	registered[channel_name] = channel

	return nil
}

func Registered() map[string]Channel {
	mt.Lock()
	defer mt.Unlock()

	return registered
}

func Factory(channel_name string, direction Direction) (channel Channel, err error) {
	mt.Lock()
	defer mt.Unlock()

	if channel_name == "" {
		return nil, fmt.Errorf("channel name can not be empty.")
	}

	channel_args := ""
	if strings.Contains(channel_name, ":") {
		parts := strings.SplitN(channel_name, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Could not parse channel name and args from '%s'.", channel_name)
		}

		channel_name = parts[0]
		channel_args = parts[1]
	}

	found := false
	if channel, found = registered[channel_name]; found == false {
		return nil, fmt.Errorf("No channel with name %s has been registered.", channel_name)
	}

	if err := channel.Setup(direction, channel_args); err != nil {
		return nil, err
	}
	if direction == INPUT_CHANNEL && channel.HasReader() == false {
		return nil, fmt.Errorf("Can't use channel '%s' for reading.", channel_name)
	} else if direction == OUTPUT_CHANNEL && channel.HasWriter() == false {
		return nil, fmt.Errorf("Can't use channel '%s' for writing.", channel_name)
	}

	return channel, nil
}
