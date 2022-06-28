/*
 * MIT License
 *
 * (C) Copyright [2019-2021] Hewlett Packard Enterprise Development LP
 *
 * Permission is hereby granted, free of charge, to any person obtaining a
 * copy of this software and associated documentation files (the "Software"),
 * to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the
 * Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included
 * in all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
 * THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
 * OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
 * ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
 * OTHER DEALINGS IN THE SOFTWARE.
 */

package main

import (
	"net/http"
)

// Fake payload of sls/v1/hardware/search

var cabSideLoad = []string{`{"Parent":"","Xname":"x1000","Type":"comptype_cabinet","TypeString":"Cabinet","Class":"Mountain","ExtraProperties":{"Network": "HMN","IP6Prefix":"fd66:0:0:0","IP4Base":"10.0.2.100/22","MACprefix":"02"}}`,
	`{"Parent":"","Xname":"x1001","Type":"comptype_cabinet","TypeString":"Cabinet","Class":"Mountain","ExtraProperties":{"Network": "HMN","IP6Prefix":"fd66:0:0:0","IP4Base":"10.10.2.100/22","MACprefix":"02"}}`,
	`{"Parent":"","Xname":"x1002","Type":"comptype_cabinet","TypeString":"Cabinet","Class":"Mountain","ExtraProperties":{"Network": "HMN","IP6Prefix":"fd66:0:0:0","IP4Base":"10.20.2.100/22"}}`, //this one tests missing macprefix
	`{"Parent":"","Xname":"x1003","Type":"comptype_cabinet","TypeString":"Cabinet","Class":"Mountain","ExtraProperties":{"Network": "HMN","IP6Prefix":"fd66:0:0:0","IP4Base":"10.30.2.100/22","MACprefix":"02"}}`}

// Expected net endpoints from MEDS' endpoint calculations

var expEP = []NetEndpoint{{name: "x1000c0b0", mac: "02:03:E8:00:00:00", hwtype: 3},
	{name: "x1001c0b0", mac: "02:03:E9:00:00:00", hwtype: 3},
	{name: "x1002c0b0", mac: "02:03:EA:00:00:00", hwtype: 3},
	{name: "x1003c0b0", mac: "02:03:EB:00:00:00", hwtype: 3},
}

var glbHttpStatus int
var hwEmpty bool = false

// Mocked SLS /search endpoint

func doHWSearch(w http.ResponseWriter, req *http.Request) {
	var payload string

	payload = "["
	if !hwEmpty {
		for ii, slstr := range cabSideLoad {
			if ii == 0 {
				payload = payload + slstr
			} else {
				payload = payload + "," + slstr
			}
		}
	}
	payload = payload + "]"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(glbHttpStatus)
	w.Write([]byte(payload))
}
