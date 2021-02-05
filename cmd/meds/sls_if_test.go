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
	"github.com/hashicorp/go-retryablehttp"
	"net/http"
	"net/http/httptest"
	"testing"
	"stash.us.cray.com/HMS/hms-certs/pkg/hms_certs"
)

// Fake payload of sls/v1/hardware/search

var cabSideLoad = []string{`{"Parent":"","Xname":"x1000","Type":"comptype_cabinet","TypeString":"Cabinet","Class":"Mountain","ExtraProperties":{"Network": "HMN","IP6Prefix":"fd66:0:0:0","IP4Base":"10.0.2.100/22","MACprefix":"02"}}`,
	`{"Parent":"","Xname":"x1001","Type":"comptype_cabinet","TypeString":"Cabinet","Class":"Mountain","ExtraProperties":{"Network": "HMN","IP6Prefix":"fd66:0:0:0","IP4Base":"10.10.2.100/22","MACprefix":"02"}}`,
	`{"Parent":"","Xname":"x1002","Type":"comptype_cabinet","TypeString":"Cabinet","Class":"Mountain","ExtraProperties":{"Network": "HMN","IP6Prefix":"fd66:0:0:0","IP4Base":"10.20.2.100/22"}}`, //this one tests missing macprefix
	`{"Parent":"","Xname":"x1003","Type":"comptype_cabinet","TypeString":"Cabinet","Class":"Mountain","ExtraProperties":{"Network": "HMN","IP6Prefix":"fd66:0:0:0","IP4Base":"10.30.2.100/22","MACprefix":"02"}}`}

// Expected net endpoints from MEDS' endpoint calculations

var expEP = []NetEndpoint{{name: "x1000c0b0", mac: "02:03:E8:00:00:00", ipv4: "10.0.2.100", ip6g: "fd66::3:e8ff:fe00:0", ip6l: "fe80::3:e8ff:fe00:0", hwtype: 3},
	{name: "x1001c0b0", mac: "02:03:E9:00:00:00", ipv4: "10.10.2.100", ip6g: "fd66::3:e9ff:fe00:0", ip6l: "fe80::3:e9ff:fe00:0", hwtype: 3},
	{name: "x1002c0b0", mac: "02:03:EA:00:00:00", ipv4: "10.20.2.100", ip6g: "fd66::3:eaff:fe00:0", ip6l: "fe80::3:eaff:fe00:0", hwtype: 3},
	{name: "x1003c0b0", mac: "02:03:EB:00:00:00", ipv4: "10.30.2.100", ip6g: "fd66::3:ebff:fe00:0", ip6l: "fe80::3:ebff:fe00:0", hwtype: 3},
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

func TestGetCabInfo(t *testing.T) {
	var endpointsNew = make([]*NetEndpoint, 0)
	var endpointsOld = make([]*NetEndpoint, 0)
	var rackInfo = RackInfo{rackList: []int{1000, 1001, 1002, 1003},
		rackIPList: []string{"10.0.2.100/22",
			"10.10.2.100/22",
			"10.20.2.100/22",
			"10.30.2.100/22"},
		ip6prefix: "fd66:0:0:0",
		macprefix: "02"}

	//Create test http server, create test client, assign to global client var
	glbHttpStatus = http.StatusOK
	retryClient := retryablehttp.NewClient()

	svr := httptest.NewServer(http.HandlerFunc(doHWSearch))
	sls = svr.URL

	rfClient = &hms_certs.HTTPClientPair{SecureClient: retryClient, InsecureClient: retryClient,}

	err := getCabInfo(&endpointsNew, rackInfo)
	if err != nil {
		t.Error("ERROR getting cabinet info via SLS:", err)
	}

	for _, ep := range endpointsNew {
		for _, xp := range expEP {
			if xp.name == ep.name {
				if *ep != xp {
					t.Errorf("NEW: Endpoint mismatch,\nexp: '%v'\ngot: '%v'\n", xp, *ep)
				}
			}
		}
	}

	////// ERROR CONDITIONS

	t.Log("TESTING ERROR CONDITIONS")

	// Bad status code from SLS

	sls = svr.URL
	glbHttpStatus = http.StatusInternalServerError
	err = getCabInfo(&endpointsOld, rackInfo)
	if err == nil {
		t.Errorf("ERROR getting cabinet via SLS didn't fail, should have.\n")
	}
	glbHttpStatus = http.StatusOK

	// No data in SLS (not an error)

	hwEmpty = true
	err = getCabInfo(&endpointsOld, rackInfo)
	if err != nil {
		t.Error("ERROR empty SLS failed:", err)
	}
	hwEmpty = false
}
