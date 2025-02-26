// Copyright (c) 2022 individual contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// <https://www.apache.org/licenses/LICENSE-2.0>
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"

	"github.com/0x5a17ed/uefi/efi/efiguid"
)

func main() {

	c := NewContext(DefaultEfiPath)

	c.Get("RPI_EFI.fd", efiguid.MustFromString("eb704011-1402-11d3-8e77-00a0c969723b"), nil)

	// _, a, err := ReadAll(c, "RPI_EFI.fd", efiguid.MustFromString("eb704011-1402-11d3-8e77-00a0c969723b"))
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// fmt.Println(a)

	// if err := BootNext.Set(c, 1); err != nil {
	// 	fmt.Println(err)
	// }

	v, err := c.VariableNames()
	if err != nil {
		fmt.Println(err)
	}

	for v.Next() {
		fmt.Println(v.Value())
	}

	bootNext := []byte{}

	_, _, err = c.Get("BootNext", efiguid.MustFromString("eb704011-1402-11d3-8e77-00a0c969723b"), bootNext)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(bootNext)

}
