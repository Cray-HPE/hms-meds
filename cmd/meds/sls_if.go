// MIT License
//
// (C) Copyright [2019-2021] Hewlett Packard Enterprise Development LP
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	sls_common "stash.us.cray.com/HMS/hms-sls/pkg/sls-common"

	base "stash.us.cray.com/HMS/hms-base"
)

// Generic Hardware data.  NOTE: this is currently replicated.
// It may move to a common package someday, but for now we'll
// make our own copy and data types.

type HMSStringType string
type CabinetType string

type GenericHardware struct {
	Parent             string        `json:"Parent"`
	Children           []string      `json:"Children,omitempty"`
	Xname              string        `json:"Xname"`
	Type               HMSStringType `json:"Type"`
	Class              CabinetType   `json:"Class"`
	TypeString         base.HMSType  `json:"TypeString"`
	ExtraPropertiesRaw interface{}   `json:"ExtraProperties,omitempty"`
}

type GenericHardwareArray []GenericHardware

// Query SLS for relevant cabinet info.  We need the cabinet XName,
// cabinet class (mountain), MACPrefix, IPv4 CIDR, IPv6 prefix.
// If --sls=xxxx or MEDS_SLS=xxx was not specified, then return
// nothing, which will cause MEDS to try the older method.

func getSLSCabInfo() ([]GenericHardware, error) {
	var jdata []GenericHardware
	var mcabs []GenericHardware
	var hcabs []GenericHardware

	//Search:  /search/hardware?type=comptype_cabinet&class=Mountain

	rsp, err := client.InsecureClient.Get(sls + "/search/hardware?type=comptype_cabinet&class=Mountain")

	if err != nil {
		log.Printf("ERROR in GET of hardware search: %v\n", err)
		return nil, err
	}
	if rsp.StatusCode != http.StatusOK {
		emsg := fmt.Sprintf("Bad error code from hardware SLS /search GET: %d/%s\n",
			rsp.StatusCode, http.StatusText(rsp.StatusCode))
		return nil, fmt.Errorf(emsg)
	}

	body, berr := ioutil.ReadAll(rsp.Body)
	if berr != nil {
		log.Printf("ERROR reading SLS /search response body (Mountain): %v\n", berr)
		return nil, berr
	}
	defer rsp.Body.Close()

	//Process the returned data

	umerr := json.Unmarshal(body, &mcabs)
	if umerr != nil {
		log.Printf("ERROR unmarshalling SLS /search response body (for Mountain): %v\n", umerr)
		return nil, umerr
	}

	//Search:  /search/hardware?type=comptype_cabinet&class=Hill

	rsp, err = client.InsecureClient.Get(sls + "/search/hardware?type=comptype_cabinet&class=Hill")

	if err != nil {
		log.Printf("ERROR in GET of hardware search: %v\n", err)
		return nil, err
	}
	if rsp.StatusCode != http.StatusOK {
		emsg := fmt.Sprintf("Bad error code from hardware SLS /search GET: %d/%s\n",
			rsp.StatusCode, http.StatusText(rsp.StatusCode))
		return nil, fmt.Errorf(emsg)
	}

	body, berr = ioutil.ReadAll(rsp.Body)
	if berr != nil {
		log.Printf("ERROR reading SLS /search response body (Hill): %v\n", berr)
		return nil, berr
	}
	defer rsp.Body.Close()

	//Process the returned data

	umerr = json.Unmarshal(body, &hcabs)
	if umerr != nil {
		log.Printf("ERROR unmarshalling SLS /search response body (for Hill): %v\n", umerr)
		return nil, umerr
	}

	for _, v := range mcabs {
		jdata = append(jdata, v)
	}

	for _, v := range hcabs {
		jdata = append(jdata, v)
	}

	return jdata, nil
}

// Get relevant cabinet information.  If SLS URL is present then query SLS.
// If not, then use the older method from cmdline arguments.  If either
// method fails, an error is returned, causing MEDS to panic.

func getCabInfo(endpoints *[]*NetEndpoint, rackInfo RackInfo) error {
	log.Printf("INFO: Gathering cabinet info from SLS.\n")

	hwList, hwerr := getSLSCabInfo()

	if hwerr != nil {
		log.Printf("ERROR, can't get cabinet list from SLS: %v\n",
			hwerr)
		return hwerr
	}
	if len(hwList) == 0 {
		log.Printf("INFO: No cabinets found in SLS.\n")
		return nil //TODO: should this be an error?
	}

	for _, cab := range hwList {
		var cb sls_common.ComptypeCabinet
		var rackIP *string
		var macPre string
		//Get ExtraProperties
		ba, baerr := json.Marshal(cab.ExtraPropertiesRaw)
		if baerr != nil {
			err := fmt.Errorf("INTERNAL ERROR, can't marshal cab props: %v\n",
				baerr)
			log.Println(err)
			return err
		}

		baerr = json.Unmarshal(ba, &cb)
		if baerr != nil {
			err := fmt.Errorf("INTERNAL ERROR, can't unmarshal cab props: %v\n",
				baerr)
			log.Println(err)
			return err
		}

		log.Printf("INFO: SLS Cab info for %s: '%s'\n", cab.Xname, string(ba))

		// Make sure the map checks out before reaching into it to avoid panic.
		hmnNetwork, networkExists := cb.Networks["cn"]["HMN"]
		if !networkExists {
			log.Printf("Cabinet doesn't have HMN network for compute nodes: %+v\n", cb)
			continue
		}

		if hmnNetwork.CIDR != "" {
			rackIP = &hmnNetwork.CIDR
		} else {
			rackIP = nil
		}

		macPre = hmnNetwork.IPv6Prefix
		if hmnNetwork.IPv6Prefix == "" {
			macPre = rackInfo.macprefix
		}
		rackNum, rerr := strconv.Atoi(cab.Xname[1:])
		if rerr != nil {
			log.Printf("INTERNAL ERROR, can't convert Xname '%s' to int: %v\n",
				cab.Xname, rerr)
			continue
		}
		*endpoints = append(*endpoints, GenerateEnvironmentalControllerEndpoints(hmnNetwork.IPv6Prefix, rackNum)...)
		*endpoints = append(*endpoints, GenerateChassisEndpoints(hmnNetwork.IPv6Prefix, rackIP, macPre, rackNum)...)

		// This can be used to print out node addresses
		if *printNodes == true {
			printHostsFormat(*endpoints)
		}
	}

	log.Printf("INFO: %d endpoints calculated for %d cabinets.\n",
		len(*endpoints), len(hwList))
	return nil
}
