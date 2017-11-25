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
	"fmt"
	"github.com/evilsocket/sg1"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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

type PastebinAPI struct {
	ApiKey  string
	UserKey string
}

func NewPastebinAPI(ApiKey, UserKey string) *PastebinAPI {
	return &PastebinAPI{
		ApiKey:  ApiKey,
		UserKey: UserKey,
	}
}

func (api *PastebinAPI) Request(page string, values url.Values) (body string, err error) {
	values.Set("api_dev_key", api.ApiKey)
	values.Set("api_user_key", api.UserKey)

	sg1.Debug("PastebinAPI.Request( %s, %s )\n", page, values)

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

func (api *PastebinAPI) GetPaste(key string) (body string, err error) {
	values := url.Values{}
	values.Set("api_paste_key", key)
	values.Set("api_option", "show_paste")

	return api.Request("api_raw.php", values)
}

func (api *PastebinAPI) parseXmlPastes(body string) []XmlPaste {
	keyParser := regexp.MustCompile("^<paste_key>(.+)</paste_key>$")
	titleParser := regexp.MustCompile("^<paste_title>SG1 (0x[a-fA-F0-9]+)</paste_title>$")
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

func (api *PastebinAPI) GetPastes() (pastes []XmlPaste, err error) {
	values := url.Values{}
	values.Set("api_option", "list")
	values.Set("api_results_limit", "1000")

	body, err := api.Request("api_post.php", values)
	if err != nil {
		return nil, err
	}

	return api.parseXmlPastes(body), nil
}

func (api *PastebinAPI) DeletePaste(paste XmlPaste) (resp string, err error) {
	values := url.Values{}
	values.Set("api_paste_key", paste.key)
	values.Set("api_option", "delete")

	return api.Request("api_post.php", values)
}

func (api *PastebinAPI) CreatePaste(paste Paste) (resp string, err error) {
	values := url.Values{}
	values.Set("api_option", "paste")
	values.Set("api_paste_code", paste.Text)
	values.Set("api_paste_name", paste.Name)
	values.Set("api_paste_private", paste.Privacy)
	values.Set("api_paste_expire_date", paste.ExpireDate)

	return api.Request("api_post.php", values)
}
