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
	"fmt"
	"sync"
)

var (
	registered = make(map[string]Module)
	mt         = &sync.Mutex{}
)

func Register(module Module) error {
	mt.Lock()
	defer mt.Unlock()

	module_name := module.Name()

	if _, found := registered[module_name]; found {
		return fmt.Errorf("Module with name %s already registered.", module_name)
	}

	if err := module.Register(); err != nil {
		return err
	}

	registered[module_name] = module

	return nil
}

func Registered() map[string]Module {
	mt.Lock()
	defer mt.Unlock()
	return registered
}

func Factory(module_name string) (module Module, err error) {
	mt.Lock()
	defer mt.Unlock()

	if module_name == "" {
		return nil, fmt.Errorf("Module name can not be empty.")
	}

	if module, found := registered[module_name]; found {
		return module, nil
	}

	return nil, fmt.Errorf("No module with name %s has been registered.", module_name)
}
