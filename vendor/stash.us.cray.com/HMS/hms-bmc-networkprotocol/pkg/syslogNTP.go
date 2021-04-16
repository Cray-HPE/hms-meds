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

package bmc_nwprotocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"stash.us.cray.com/HMS/hms-certs/pkg/hms_certs"
)

// This package provides an interface for sending Network Protocol stuff
// to Mountain BMCs.  Initially this is Syslog and NTP info, but can
// be expanded if need be.
//
// Typical usage:
//   In main, call bmc_nwprotocol.Init(syslogTarg,ntpTarg,rfSuffix)
//
// When wanting to send NW protocol info to a BMC:
//   bmc_nwprotocol.SetXNameNTPSyslog(bmcAddr,username,pw)

// This is used for the Init() function to populate the innards of this
// library.

type NWPData struct {
	CAChainURI    string //pathname or hms_certs.VaultCAChainURI
	NTPSpec       string //ip_or_hostname:port
	SyslogSpec    string //ip_or_hostname:port
	SSHKey        string
	SSHConsoleKey string
	BootOrder     []string
}

type SyslogData struct {
	ProtocolEnabled bool     `json:"ProtocolEnabled,omitempty"`
	SyslogServers   []string `json:"SyslogServers,omitempty"`
	Transport       string   `json:"Transport,omitempty"`
	Port            int      `json:"Port,omitempty"`
}

type SSHAdminData struct {
	AuthorizedKeys string `json:"AuthorizedKeys"`
}

type OemData struct {
	Syslog     *SyslogData   `json:"Syslog,omitempty"`
	SSHAdmin   *SSHAdminData `json:"SSHAdmin,omitempty"`
	SSHConsole *SSHAdminData `json:"SSHConsole,omitempty"`
}

type NTPData struct {
	NTPServers      []string `json:"NTPServers,omitempty"`
	ProtocolEnabled bool     `json:"ProtocolEnabled,omitempty"`
	Port            int      `json:"Port,omitempty"`
}

type RedfishNWProtocol struct {
	valid bool     //not marshallable
	Oem   *OemData `json:"Oem,omitempty"`
	NTP   *NTPData `json:"NTP,omitempty"`
}

var redfishNPSuffix string //comes from cmdline or env
var http_client *hms_certs.HTTPClientPair
var serviceName string

// Split a NTP or syslog "uri" spec.  Format will be
//    <ip_or_hostname,ip_or_hostname...:port>

func splitNWSpec(spec string) ([]string, int, error) {
	var rstrs []string
	toks := strings.Split(spec, ":")
	if len(toks) < 2 {
		return rstrs, -1, fmt.Errorf("Can't split target specification '%s', incorrect format.\n", spec)
	}
	ips := strings.Split(toks[0], ",")
	pint, err := strconv.Atoi(toks[1])
	if err != nil {
		return rstrs, -1, fmt.Errorf("Can't convert port of target specification '%s', incorrect format.\n", spec)
	}
	return ips, pint, nil
}

func CopyRFNetworkProtocol(src *RedfishNWProtocol) RedfishNWProtocol {
	var ret RedfishNWProtocol

	ret.valid = src.valid
	if (src.NTP != nil) {
		ret.NTP = &NTPData{ProtocolEnabled: src.NTP.ProtocolEnabled,
		                   Port: src.NTP.Port,}
		if (len(src.NTP.NTPServers) != 0) {
			ret.NTP.NTPServers = append(src.NTP.NTPServers[:0:0],src.NTP.NTPServers...)
		}
	}
	if (src.Oem != nil) {
		ret.Oem = &OemData{}
		if (src.Oem.Syslog != nil) {
			ret.Oem.Syslog = &SyslogData{ProtocolEnabled: src.Oem.Syslog.ProtocolEnabled,
			                             Transport: src.Oem.Syslog.Transport,
			                             Port: src.Oem.Syslog.Port,}
			if (len(src.Oem.Syslog.SyslogServers) != 0) {
				ret.Oem.Syslog.SyslogServers = append(src.Oem.Syslog.SyslogServers[:0:0],src.Oem.Syslog.SyslogServers...)
			}
		}
		if (src.Oem.SSHAdmin != nil) {
			ret.Oem.SSHAdmin = &SSHAdminData{AuthorizedKeys:src.Oem.SSHAdmin.AuthorizedKeys,}
		}
		if (src.Oem.SSHConsole != nil) {
			ret.Oem.SSHConsole = &SSHAdminData{AuthorizedKeys:src.Oem.SSHConsole.AuthorizedKeys,}
		}
	}

	return ret
}

// Given specification of NTP and/or syslog endpoints,
// create the data structs used by Redfish on the controllers to
// set NTP and syslog data.  This is typically called once, by main().
//
// nwpData:  Contains the specifications for Redfish NWP data parameters
// rfSuffix: Formatted /redfish/v1/Managers/BMC/NetworkProtocol or similar
// Return:   Populated RedfishNWProtocol struct, and error data on failure

func Init(nwpData NWPData, rfSuffix string) (RedfishNWProtocol, error) {
	var nwProtoInfo RedfishNWProtocol
	var errstrs string
	var err error

	if (serviceName == "") {
		serviceName,err = os.Hostname()
		if (err != nil) {
			serviceName = "NWP"	//Lame!  But, at least gives some indication.
		}
	}
	hms_certs.InitInstance(nil,serviceName)

	if (http_client == nil) {
		http_client,err = hms_certs.CreateHTTPClientPair(nwpData.CAChainURI,17)
		if (err != nil) {
			return nwProtoInfo,fmt.Errorf("ERROR creating TLS cert-enabled HTTP client: %v",
				err)
		}
	}

	redfishNPSuffix = rfSuffix
	nwProtoInfo.valid = true
	if (nwpData.SyslogSpec != "") || (nwpData.SSHKey != "") ||
		(nwpData.SSHConsoleKey != "") {
		nwProtoInfo.Oem = new(OemData)
	}

	if nwpData.SyslogSpec != "" {
		var iparr []string
		sarr, pint, err := splitNWSpec(nwpData.SyslogSpec)
		if err != nil {
			errstrs += fmt.Sprintf("syslog spec: '%v';  ", err)
		} else {
			nwProtoInfo.Oem.Syslog = new(SyslogData)
			nwProtoInfo.Oem.Syslog.ProtocolEnabled = true
			nwProtoInfo.Oem.Syslog.Transport = "udp"

			//If the host of this spec is not an IP addr, convert it to one.
			//TODO: this is IPV4 only for now.
			for ii := 0; ii < len(sarr); ii++ {
				// check if the name lookup works - just log if there is a problem
				//  as the name resolution may be updated later
				_, iperr := net.LookupIP(sarr[ii])
				if iperr != nil {
					errstrs += fmt.Sprintf("syslog target [%s] can't convert to IP addr: '%v';  ",
						sarr[ii], iperr)
				}
				iparr = append(iparr, sarr[ii])
			}

			nwProtoInfo.Oem.Syslog.SyslogServers = iparr
			nwProtoInfo.Oem.Syslog.Port = pint
			log.Printf("INFO: Mountain controller syslog forwarding target: %v:%d\n", iparr, pint)
		}
	}

	if nwpData.NTPSpec != "" {
		var iparr []string
		sarr, pint, err := splitNWSpec(nwpData.NTPSpec)
		if err != nil {
			errstrs += fmt.Sprintf("NTP spec: '%v';  ", err)
		} else {
			nwProtoInfo.NTP = new(NTPData)
			nwProtoInfo.NTP.ProtocolEnabled = true

			//If the host of this spec is not an IP addr, convert it to one.
			//TODO: this is IPV4 only for now.
			for ii := 0; ii < len(sarr); ii++ {
				ip, iperr := net.LookupIP(sarr[ii])
				if iperr == nil {
					iparr = append(iparr, ip[0].String())
				} else {
					errstrs += fmt.Sprintf("NTP target [%s] can't convert to IP addr: '%v';  ",
						sarr[ii], iperr)
				}
			}

			nwProtoInfo.NTP.NTPServers = iparr
			nwProtoInfo.NTP.Port = pint
			log.Printf("INFO: Mountain controller NTP server: %v:%d\n", iparr, pint)
		}
	}

	if nwpData.SSHKey != "" {
		nwProtoInfo.Oem.SSHAdmin = &SSHAdminData{AuthorizedKeys: nwpData.SSHKey}
	}

	if nwpData.SSHConsoleKey != "" {
		nwProtoInfo.Oem.SSHConsole = &SSHAdminData{AuthorizedKeys: nwpData.SSHConsoleKey}
	}

	//TODO: boot order, once it's figured out.

	if len(errstrs) > 0 {
		return nwProtoInfo, fmt.Errorf(errstrs)
	}

	return nwProtoInfo, nil
}

// Same init process, but allows for setting the service name for User-Agent
// purposes.
//
// nwpData:  Contains the specifications for Redfish NWP data parameters
// rfSuffix: Formatted /redfish/v1/Managers/BMC/NetworkProtocol or similar
// svcName:  Caller's application name.
// Return:   Populated RedfishNWProtocol struct, and error data on failure

func InitInstance(nwpData NWPData, rfSuffix string, svcName string) (RedfishNWProtocol, error) {
	serviceName = svcName
	return Init(nwpData, rfSuffix)
}

// Set NTP and syslog endpoint data for a newly-discovered mountain
// controller.

func SetXNameNWPInfo(nwProtoInfo RedfishNWProtocol, targAddress, RFUsername, RFPassword string) error {
	var url string

	if !nwProtoInfo.valid {
		return fmt.Errorf("NW proto info valid bit not set.")
	}

	ba, baerr := json.Marshal(nwProtoInfo)
	if baerr != nil {
		rerr := fmt.Errorf("Can't marshal NW Protcol info: %v", baerr)
		return rerr
	}

	//TEST TEST TEST
	//log.Printf("INFO: Data being sent to '%s': '%s'\n",targAddress,string(ba))
	//TEST TEST TEST

	//Since the golang http client doesn't do PUT or PATCH, we have
	//to do this the hard way (gotta do a PATCH)

	//hack for unit testability: if redfishNPSuffix == __test__, just
	//use 'targAddress' parameter as the URL.

	if redfishNPSuffix == "__test__" {
		url = targAddress
	} else {
		url = "https://" + targAddress + redfishNPSuffix
	}
	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(ba))
	req.SetBasicAuth(RFUsername, RFPassword)
	req.Header.Set("Content-Type", "application/json")
	rsp, rsperr := http_client.Do(req)

	if rsperr != nil {
		rerr := fmt.Errorf("ERROR sending NTP/syslog info to '%s': %v",
			targAddress, rsperr)
		return rerr
	}

	if (rsp.Body != nil) {
		_,_ = ioutil.ReadAll(rsp.Body)
		defer rsp.Body.Close()
	}

	if (rsp.StatusCode != http.StatusOK) && (rsp.StatusCode != http.StatusNoContent) {
		rerr := fmt.Errorf("ERROR sending NTP/syslog info to '%s', status code %d",
			targAddress, rsp.StatusCode)
		return rerr
	}

	log.Printf("INFO: Successfully sent syslog/NTP data to '%s'\n", targAddress)
	return nil
}
