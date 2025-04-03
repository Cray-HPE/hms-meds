/*
 *
 *  MIT License
 *
 *  (C) Copyright 2019-2022,2025 Hewlett Packard Enterprise Development LP
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a
 *  copy of this software and associated documentation files (the "Software"),
 *  to deal in the Software without restriction, including without limitation
 *  the rights to use, copy, modify, merge, publish, distribute, sublicense,
 *  and/or sell copies of the Software, and to permit persons to whom the
 *  Software is furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included
 *  in all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
 *  THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
 *  OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
 *  ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
 *  OTHER DEALINGS IN THE SOFTWARE.
 *
 */

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	sls_common "github.com/Cray-HPE/hms-sls/v2/pkg/sls-common"
	"github.com/Cray-HPE/hms-xname/xnametypes"
)

// Query SLS for relevant cabinet info.  We need the cabinet XName,
// cabinet class (mountain), MACPrefix, IPv4 CIDR, IPv6 prefix.
// If --sls=xxxx or MEDS_SLS=xxx was not specified, then return
// nothing, which will cause MEDS to try the older method.

func getSLSCabInfo() ([]sls_common.GenericHardware, error) {
	var jdata []sls_common.GenericHardware
	var mcabs []sls_common.GenericHardware
	var hcabs []sls_common.GenericHardware
	var body []byte
	var berr error

	//Search:  /search/hardware?type=comptype_cabinet&class=Mountain

	rsp, err := client.InsecureClient.Get(sls + "/search/hardware?type=comptype_cabinet&class=Mountain")

	if err != nil {
		log.Printf("ERROR in GET of hardware search: %v\n", err)
		return nil, err
	}

	if rsp.Body != nil {
		body, berr = ioutil.ReadAll(rsp.Body)
		defer rsp.Body.Close()
	}

	if rsp.StatusCode != http.StatusOK {
		emsg := fmt.Sprintf("Bad error code from hardware SLS /search GET: %d/%s\n",
			rsp.StatusCode, http.StatusText(rsp.StatusCode))
		return nil, fmt.Errorf("%s", emsg)
	}

	if berr != nil {
		log.Printf("ERROR reading SLS /search response body (Mountain): %v\n", berr)
		return nil, berr
	}

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

	if rsp.Body != nil {
		body, berr = ioutil.ReadAll(rsp.Body)
		defer rsp.Body.Close()
	}

	if rsp.StatusCode != http.StatusOK {
		emsg := fmt.Sprintf("Bad error code from hardware SLS /search GET: %d/%s\n",
			rsp.StatusCode, http.StatusText(rsp.StatusCode))
		return nil, fmt.Errorf("%s", emsg)
	}

	if berr != nil {
		log.Printf("ERROR reading SLS /search response body (Hill): %v\n", berr)
		return nil, berr
	}

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

func getSLSCabinetChassis(cabinetXname string) ([]sls_common.GenericHardware, error) {
	if xnametypes.GetHMSType(cabinetXname) != xnametypes.Cabinet {
		return nil, fmt.Errorf("provided xname is not a cabinet: %v", cabinetXname)
	}

	// Search:  /search/hardware?type=comptype_chassis&parent=x1000
	var body []byte
	var berr error
	rsp, err := client.InsecureClient.Get(sls + fmt.Sprintf("/search/hardware?type=comptype_chassis&parent=%s", cabinetXname))
	if err != nil {
		log.Printf("ERROR in GET of hardware search: %v\n", err)
		return nil, err
	}

	if rsp.Body != nil {
		body, berr = ioutil.ReadAll(rsp.Body)
		defer rsp.Body.Close()
	}

	if rsp.StatusCode != http.StatusOK {
		emsg := fmt.Sprintf("Bad error code from hardware SLS /search GET: %d/%s\n",
			rsp.StatusCode, http.StatusText(rsp.StatusCode))
		return nil, fmt.Errorf("%s", emsg)
	}

	if berr != nil {
		log.Printf("ERROR reading SLS /search response body (for chassis of cabinet %v): %v\n", cabinetXname, berr)
		return nil, berr
	}

	// Process the returned data
	var unprocessedCabinetChassis []sls_common.GenericHardware
	umerr := json.Unmarshal(body, &unprocessedCabinetChassis)
	if umerr != nil {
		log.Printf("ERROR unmarshalling SLS /search response body (for chassis of cabinet %v): %v\n", cabinetXname, umerr)
		return nil, umerr
	}

	// Filter out any river chassis returned in the query
	var cabinetChassis []sls_common.GenericHardware
	for _, chassis := range unprocessedCabinetChassis {
		if chassis.Class == sls_common.ClassRiver {
			continue
		}

		cabinetChassis = append(cabinetChassis, chassis)
	}

	return cabinetChassis, nil
}
