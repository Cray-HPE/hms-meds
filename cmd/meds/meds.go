/*
 * MIT License
 *
 * (C) Copyright [2019-2022] Hewlett Packard Enterprise Development LP
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
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	base "github.com/Cray-HPE/hms-base/v2"
	dns_dhcp "github.com/Cray-HPE/hms-dns-dhcp/pkg"
	sls_common "github.com/Cray-HPE/hms-sls/pkg/sls-common"
	"github.com/Cray-HPE/hms-smd/pkg/sm"
	"github.com/Cray-HPE/hms-xname/xnametypes"

	"github.com/Cray-HPE/hms-meds/internal/model"

	compcreds "github.com/Cray-HPE/hms-compcredentials"
	sstorage "github.com/Cray-HPE/hms-securestorage"

	bmc_nwprotocol "github.com/Cray-HPE/hms-bmc-networkprotocol/pkg"
	"github.com/Cray-HPE/hms-certs/pkg/hms_certs"
)

// A general understanding of the hardware in a Mountain rack is necessary
// to fully understand and appreciate the way items in this are generated.
//
// At the top level, each cabinet has 2 Environmental controllers (eC) and
// 8 chassis (each of which has a chassis controller (cC)). Each chassis
// then has eight slots, each slot contains 2 node cards (nC)

// MTN_eC_COUNT is the number of Environmental Controllers per rack.
const MTN_eC_COUNT = 2

// MTN_CHASSIS_COUNT is the number of chasses per rack.
const MTN_CHASSIS_COUNT = 8

// MTN_SWITCH_COUNT is the number of switches in a chassis
const MTN_SWITCH_COUNT = 8

// MTN_nC_PER_SLOT is the number of node cards in each slot.
const MTN_nC_PER_SLOT = 2

const MAC_PREFIX = "02"

type HSMNotification struct {
	ID                 string `json:"ID"`
	FQDN               string `json:"FQDN,omitempty"`
	Hostname           string `json:"Hostname,omitempty"`
	IPAddress          string `json:"IPAddress,omitempty"`
	User               string `json:"User,omitempty"`
	Password           string `json:"Password,omitempty"`
	MACAddr            string `json:"MACAddr,omitempty"`
	RediscoverOnUpdate bool   `json:"RediscoverOnUpdate,omitempty"`
	Enabled            *bool  `json:"Enabled,omitempty"` //need to set a default
}

type HSMNotificationArray struct {
	RedfishEndpoints []HSMNotification `json:"RedfishEndpoints"`
}

type NetEndpoint struct {
	name        string
	mac         string
	hwtype      int
	HSMPresence HSMEndpointPresence
	HSMPresLock sync.Mutex
	QuitChannel chan struct{}
}

type RackList []int
type RackIPList []string

type NetEndpointList struct {
	ec []*NetEndpoint
	cc []*NetEndpoint
	nc []*NetEndpoint
	sc []*NetEndpoint
	nd []*NetEndpoint
}

type EndpointType int

const (
	TYPE_ENV_CONTROLLER = iota
	TYPE_NODE_CARD
	TYPE_SWITCH_CARD
	TYPE_CHASSIS
)

type HSMEndpointPresence int

// Used only for passing rack info to initial discovery func

type RackInfo struct {
	rackList   RackList
	rackIPList RackIPList
	ip6prefix  string
	macprefix  string
}

const (
	PRESENCE_PRESENT = iota
	PRESENCE_NOT_PRESENT
)

var EndpointTypeToString map[int]string = map[int]string{
	TYPE_ENV_CONTROLLER: "Environmental Controller",
	TYPE_NODE_CARD:      "Node Card",
	TYPE_SWITCH_CARD:    "Switch Card",
	TYPE_CHASSIS:        "Chassis",
}

var HSMEndpointPresenceToString map[HSMEndpointPresence]string = map[HSMEndpointPresence]string{
	PRESENCE_PRESENT:     "present",
	PRESENCE_NOT_PRESENT: "not present",
}

var serviceName string
var hsm string
var sls string
var printNodesUnder bool = false
var printNodes *bool = &printNodesUnder
var defUser string
var defPass string
var defSSHKey string
var hms_ca_uri string
var logInsecFailover = true
var clientTimeout = 5
var maxInitialHSMSyncAttempts int

// The HSM Credentials store
var hcs *compcreds.CompCredStore
var syslogTarg, ntpTarg string
var syslogTargUseIP, ntpTargUseIP bool
var smnTimeoutSecs int = 10
var redfishNPSuffix string
var debugLevel int = 0
var rfNWPStatic bmc_nwprotocol.RedfishNWProtocol

var dhcpdnsClient dns_dhcp.DNSDHCPHelper

var client *hms_certs.HTTPClientPair
var rfClient *hms_certs.HTTPClientPair
var rfClientLock sync.RWMutex

// These control the timing of checkin threads (the threads which monitor individual endpoints)
// When they start they wait up to startupVariableWaitMax seconds.
// Then between checks they wait checkupFixedWait ± checkupVariableWaitMax seconds
var checkupVariableWaitMax = 5                // In seconds - the maximum variable wait between checkups
var checkupFixedWait = 30                     // In seconds - how long we must wait between checkups on each item
var startupVariableWaitMax = checkupFixedWait // In seconds - the maximum each checkup thread waits on start

var credStorage *model.MedsCredStore

// Variables for tracking what's around/available
// List of net endpoints by the cabinet they belong to
// We'll need this for now, because we have to add/remove endpoints based on cabinet presense
var activeCabinets map[string][]*NetEndpoint = make(map[string][]*NetEndpoint)

// A full list of endpoints with no cabinet reference
var activeEndpoints map[string]*NetEndpoint = make(map[string]*NetEndpoint)

// A full list of redfish endpoints known in HSM
var hsmRedfishEndpointsCache map[string]HSMNotification
var hsmRedfishEndpointsCacheLock sync.Mutex

// Lock for modifying the above two structures
var activeEndpointsLock sync.Mutex

// Implements the flag.Value String() method
func (r *RackList) String() string {
	var rackText string
	for _, rackNumber := range *r {
		rackText += ", " + fmt.Sprintf("%d", rackNumber)
	}
	return rackText
}

// Set implements the flag.Value Set() method
func (r *RackList) Set(val string) error {
	i, err := strconv.Atoi(val)
	if err != nil {
		return err
	}
	*r = append(*r, i)
	return nil
}

// Implements the flag.Value String() method
func (r *RackIPList) String() string {
	var rackText string
	for i := range *r {
		rackText += ", " + string((*r)[i])
	}
	return rackText
}

// Set implements the flag.Value Set() method
func (r *RackIPList) Set(val string) error {
	*r = append(*r, val)
	return nil
}

func GenerateMAC(mp string, rack int, chassis int, slt int, idx int) string {
	return fmt.Sprintf("%s:%02X:%02X:%02X:%02X:%02X", mp,
		(rack>>8)&0xFF, rack&0xFF, chassis&0xFF, slt&0xFF, (idx<<4)&0xFF)
}

func GenerateMACnC(mp string, rack int, chassis int, slt int, idx int) string {
	return GenerateMAC(mp, rack, chassis, slt+48, idx)
}

func GenerateMACsC(mp string, rack int, chassis int, slt int) string {
	return GenerateMAC(mp, rack, chassis, slt+96, 0)

}

func GenerateMACcC(mp string, rack int, chassis int) string {
	return GenerateMAC(mp, rack, chassis, 0, 0)
}

// GenerateEnvironmentalControllerEndpoints generates the Environmental
//  Controller (eC) entries for a given rack.
// Parameters:
// - ip6prefix (string): The IPv6 address prefix to use.
// - rack (int): The number of the rack to generate the Env
// Returns:
// - []NetEndpoint: a slice of NetEndpoints representing the eCs available
//   in this rack
func GenerateEnvironmentalControllerEndpoints(rack int) []*NetEndpoint {
	// eC is a special snowflake with respect to address assignment.
	ret := make([]*NetEndpoint, 0)

	for i := 0; i < MTN_eC_COUNT; i++ {
		ec := new(NetEndpoint)
		ec.name = fmt.Sprintf("x%de%d", rack, i)
		ec.hwtype = TYPE_ENV_CONTROLLER
		ec.HSMPresence = PRESENCE_NOT_PRESENT
		ret = append(ret, ec)
	}
	return ret
}

// GenerateNodeCardEndpoints builds the Node Card (nC) entries for a specific slot.
func GenerateNodeCardEndpoints(macprefix string, rack int, chassis int, slot int) []*NetEndpoint {
	ret := make([]*NetEndpoint, 0)

	for nc := 0; nc < MTN_nC_PER_SLOT; nc++ {
		ep := new(NetEndpoint)
		ep.name = fmt.Sprintf("x%dc%ds%db%d", rack, chassis, slot, nc)
		ep.mac = GenerateMACnC(macprefix, rack, chassis, slot, nc)
		ep.hwtype = TYPE_NODE_CARD
		ep.HSMPresence = PRESENCE_NOT_PRESENT
		ret = append(ret, ep)
	}
	return ret
}

func GenerateSwitchCardEndpoints(macprefix string, rack int, chassis int) []*NetEndpoint {
	endpoints := make([]*NetEndpoint, 0)

	for card := 0; card < MTN_SWITCH_COUNT; card++ {

		ep := new(NetEndpoint)
		ep.name = fmt.Sprintf("x%dc%dr%db0", rack, chassis, card)
		ep.mac = GenerateMACsC(macprefix, rack, chassis, card)
		ep.hwtype = TYPE_SWITCH_CARD
		ep.HSMPresence = PRESENCE_NOT_PRESENT
		endpoints = append(endpoints, ep)
		// Use "variadic slice append" notation. Go figure...
		// (This presents the slice to be appended as a list of
		// variadic arguments to the append function)
		endpoints = append(endpoints, GenerateNodeCardEndpoints(
			macprefix, rack, chassis, card)...)
	}

	return endpoints
}

//
func GenerateChassisEndpoints(macprefix string, rack int) []*NetEndpoint {
	endpoints := make([]*NetEndpoint, 0)

	for chassis := 0; chassis < MTN_CHASSIS_COUNT; chassis++ {
		cc := new(NetEndpoint)
		cc.name = fmt.Sprintf("x%dc%db0", rack, chassis)
		cc.mac = GenerateMACcC(macprefix, rack, chassis)
		cc.hwtype = TYPE_CHASSIS
		cc.HSMPresence = PRESENCE_NOT_PRESENT
		endpoints = append(endpoints, cc)

		endpoints = append(endpoints, GenerateSwitchCardEndpoints(
			macprefix, rack, chassis)...)
	}

	return endpoints
}

func patchXNameEnabled(xname string, enabled bool) *error {
	var strbody []byte

	payload := HSMNotification{
		ID:      xname,
		Enabled: &enabled,
		// Match the 'enabled' bool so HSM will rediscover only when
		// we are setting the redfishEndpoint to 'Enabled'.
		RediscoverOnUpdate: enabled,
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WARNING: Could not encode JSON for %s: %v (%v)", xname, err, payload)
		return &err
	}

	log.Printf("DEBUG: PATCH to %s/Inventory/RedfishEndpoints/%s", hsm, xname)

	req, err := http.NewRequest(http.MethodPatch, hsm+"/Inventory/RedfishEndpoints/"+xname, bytes.NewReader(rawPayload))
	req.Header.Add("Content-Type", "application/json")
	base.SetHTTPUserAgent(req, serviceName)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("WARNING: Unable to patch %s: %v", xname, err)
		return &err
	}

	if resp.Body != nil {
		strbody, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}

	if resp.StatusCode == 200 {
		log.Printf("INFO: Successfully patched %s", xname)
	} else {
		log.Printf("WARNING: An error occurred patching %s: %s %v", xname, resp.Status, string(strbody))
		rerr := fmt.Errorf("Unable to patch information for %s to HSM: %d\n%s", xname, resp.StatusCode, string(strbody))
		return &rerr
	}
	return nil
}

func patchXnameFQDN(xname, fqdn, hostname string) error {
	var strbody []byte

	payload := HSMNotification{
		ID:       xname,
		FQDN:     fqdn,
		Hostname: hostname,
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WARNING: Could not encode JSON for %s: %v (%v) with FQDN (%s) and Hostname (%s)", xname, err, payload, fqdn, hostname)
		return err
	}

	log.Printf("DEBUG: PATCH to %s/Inventory/RedfishEndpoints/%s", hsm, xname)

	req, err := http.NewRequest(http.MethodPatch, hsm+"/Inventory/RedfishEndpoints/"+xname, bytes.NewReader(rawPayload))
	req.Header.Add("Content-Type", "application/json")
	base.SetHTTPUserAgent(req, serviceName)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("WARNING: Unable to patch %s: %v", xname, err)
		return err
	}

	if resp.Body != nil {
		strbody, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}

	if resp.StatusCode == 200 {
		log.Printf("INFO: Successfully patched %s", xname)
	} else {
		log.Printf("WARNING: An error occurred patching %s: %s %v", xname, resp.Status, string(strbody))
		rerr := fmt.Errorf("Unable to patch information for %s to HSM: %d\n%s", xname, resp.StatusCode, string(strbody))
		return rerr
	}
	return nil
}

func notifyXnamePresent(node NetEndpoint, address string) *error {
	perNodeCred, err := hcs.GetCompCred(node.name)
	if err != nil {
		log.Printf("WARNING: Unable to retrieve key %s from vault: %s", node.name, err)
		return &err
	}

	// If we get nothing back from Vault then we need to push something in.
	if perNodeCred.Username == "" || perNodeCred.Password == "" {
		// Grab the global credentails
		globalCreds, err := credStorage.FindGlobalCredentials()
		if err != nil || len(globalCreds.Username) == 0 {
			if len(defUser) != 0 {
				log.Printf("WARNING: Unable to retrieve MEDS global credentials (err: %s) or retrieved credentials are "+
					"empty, using defaults", err)
				globalCreds = model.MedsCredentials{
					Username: defUser,
					Password: defPass,
				}
			} else {
				err = fmt.Errorf("Unable to retrieve MEDS global credentials (err: %s) or retrieved credentials are "+
					"empty. No defaults are set, so refusing to continue adding Xname", err)
				log.Printf("WARNING: %s", err)
				return &err
			}
		}

		// Push in the default creds into vault
		perNodeCred.Xname = node.name
		perNodeCred.Username = globalCreds.Username
		perNodeCred.Password = globalCreds.Password

		log.Printf("INFO: No creds exist for %s in vault, setting it to the MEDS global defaults", node.name)

		err = hcs.StoreCompCred(perNodeCred)
		if err != nil {
			// If we fail to store credentials in vault, we'll lose the
			// credentials and the component endpoints associated with
			// them will still be successfully in the database.
			log.Printf("Failed to store credentials for %s in Vault - %s", node.name, err)
		}
	}

	bmcCreds, err := credStorage.FindBMCSSHCredentials(node.name)
	if err != nil || len(bmcCreds.Username) == 0 {
		log.Printf("WARNING: Unable to retrieve MEDS SSH credentials (err: %s) or retrieved credentials are "+
			"empty, using defaults", err)
		bmcCreds = model.MedsSSHCredentials{
			Username:      defUser,
			Password:      defPass,
			AuthorizedKey: defSSHKey,
		}
	}

	tmpBMCCreds := bmc_nwprotocol.CopyRFNetworkProtocol(&rfNWPStatic)
	if tmpBMCCreds.Oem == nil {
		tmpBMCCreds.Oem = &bmc_nwprotocol.OemData{}
	}
	if bmcCreds.AuthorizedKey != "" {
		tmpBMCCreds.Oem.SSHAdmin = &bmc_nwprotocol.SSHAdminData{AuthorizedKeys: bmcCreds.AuthorizedKey}
		tmpBMCCreds.Oem.SSHConsole = &bmc_nwprotocol.SSHAdminData{AuthorizedKeys: bmcCreds.AuthorizedKey}
	} else {
		tmpBMCCreds.Oem.SSHAdmin = nil
		tmpBMCCreds.Oem.SSHConsole = nil
	}

	rfClientLock.RLock()
	nstError := bmc_nwprotocol.SetXNameNWPInfo(tmpBMCCreds, address, perNodeCred.Username, perNodeCred.Password)
	rfClientLock.RUnlock()

	hsmError := notifyHSMXnamePresent(node, address)

	if (hsmError != nil) || (nstError != nil) {
		finalError := fmt.Errorf("%v %v", hsmError, nstError)
		return &finalError
	}
	return nil
}

func notifyHSMXnamePresent(node NetEndpoint, address string) *error {
	var strbody []byte

	// No longer include User and Password (set to blank) to signal HSM to pull from Vault
	payload := HSMNotification{
		ID:                 node.name,
		FQDN:               node.name,
		User:               "", // blank to pull from Vault
		Password:           "", // blank to pull from Vault
		MACAddr:            node.mac,
		RediscoverOnUpdate: true,
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WARNING: Could not encode JSON for %s: %v (%v)", node.name, err, payload)
		return &err
	}

	log.Printf("DEBUG: POST to %s/Inventory/RedfishEndpoints with %s", hsm, string(rawPayload))

	url := hsm + "/Inventory/RedfishEndpoints"
	req, qerr := http.NewRequest(http.MethodPost, url, bytes.NewReader(rawPayload))
	if qerr != nil {
		log.Printf("WARNING: Unable to create HTTP request for %s: %v",
			node.name, qerr)
		return &qerr
	}
	req.Header.Add("Content-Type", "application/json")
	base.SetHTTPUserAgent(req, serviceName)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("WARNING: Unable to send information for %s: %v", node.name, err)
		return &err
	}

	if resp.Body != nil {
		strbody, _ = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
	}

	if resp.StatusCode == 201 {
		log.Printf("INFO: Successfully created %s", node.name)
	} else if resp.StatusCode == 409 {
		log.Printf("INFO: %s alredy present; patching instead", node.name)
		return patchXNameEnabled(node.name, true)
	} else {
		log.Printf("WARNING: An error occurred uploading %s: %s %v", node.name, resp.Status, string(strbody))
		rerr := errors.New("Unable to upload information for " + node.name + " to HSM: " + fmt.Sprint(resp.StatusCode) + "\n" + string(strbody))
		return &rerr
	}
	log.Printf("INFO: Successfully added %s to HSM", node.name)
	return nil
}

func notifyHSMXnameNotPresent(node NetEndpoint) *error {
	log.Printf("DEBUG: Would remove %s, but MEDS no longer marks redfishEndpoints as disabled. This message is purely for your information; MEDS is operating as expected.", node.name)

	return nil
}

func queryHSMState() error {
	endpoints := activeEndpoints
	// Lock the presence field for all endpoints so other
	// functions that might modify this field won't.
	for _, ep := range endpoints {
		ep.HSMPresLock.Lock()
		defer ep.HSMPresLock.Unlock()
	}

	log.Printf("DEBUG: GET from %s/Inventory/RedfishEndpoints", hsm)

	url := hsm + "/Inventory/RedfishEndpoints"
	req, qerr := http.NewRequest(http.MethodGet, url, nil)
	if qerr != nil {
		log.Printf("WARNING: Unable to create HTTP request for HSM query: %v",
			qerr)
		return qerr
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("WARNING: Unable to get RedfishEndpoints from HSM: %v", err)
		return err
	}

	if resp.Body == nil {
		emsg := fmt.Errorf("No response body from querying HSM for RedfishEndpoints.")
		log.Printf("WARNING: %v", emsg)
		return emsg
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("WARNING: Unable to read HTTP body while querying HSM for RedfishEndpoints: %v", err)
		return err
	}

	if resp.StatusCode == 200 {
		rfEPs := new(HSMNotificationArray)
		err := json.Unmarshal(bodyBytes, rfEPs)
		if err != nil {
			log.Printf("WARNING: Unable to unmarshal HSM data: %v", err)
			return err
		}
		rfEPMap := make(map[string]HSMNotification, 0)
		for _, rfEP := range rfEPs.RedfishEndpoints {
			rfEPMap[rfEP.ID] = rfEP
		}
		for _, ep := range endpoints {
			rfEP, ok := rfEPMap[ep.name]
			if !ok {
				// Redfish Endpoint was in the HSM inventory, but no longer present. ie Deleted
				if ep.HSMPresence != PRESENCE_NOT_PRESENT {
					log.Printf("DEBUG: %s is now not present in HSM", ep.name)
				}
				ep.HSMPresence = PRESENCE_NOT_PRESENT
			} else if rfEP.Enabled != nil && *(rfEP.Enabled) != true {
				// Redfish endpoint is present in HSM inventory, but has been manually marked disabled
				// MEDS treats this as if the ENDPOINT is not present/
				// present and set false
				if ep.HSMPresence != PRESENCE_NOT_PRESENT {
					log.Printf("DEBUG: %s is now not present in HSM", ep.name)
				}
				ep.HSMPresence = PRESENCE_NOT_PRESENT
			} else {
				// Redfish endpoint is present within HSM inventory and enabled
				// present and set true OR flag not present
				if ep.HSMPresence != PRESENCE_PRESENT {
					log.Printf("DEBUG: %s is now present in HSM", ep.name)
				}
				ep.HSMPresence = PRESENCE_PRESENT
			}
		}

		// Update HSM Redfish endpoint cache
		hsmRedfishEndpointsCacheLock.Lock()
		hsmRedfishEndpointsCache = rfEPMap
		hsmRedfishEndpointsCacheLock.Unlock()

		return nil
	}

	// else ...
	log.Printf("WARNING: Error occurred looking up RedfishEndpoints in HSM (code %d):\n%s", resp.StatusCode, string(bodyBytes))
	rerr := errors.New("Unable to retrieve status from HSM: " + fmt.Sprint(resp.StatusCode) + "\n" + string(bodyBytes))
	return rerr
}

func queryNetworkStatusViaAddress(address string) (HSMEndpointPresence, *error) {
	//Redfish operation; try validated HTTP first, then fail over to un-validated.
	rfClientLock.RLock()
	resp, err := rfClient.Get("https://" + address + "/redfish/v1/")
	rfClientLock.RUnlock()

	if err != nil {
		return PRESENCE_NOT_PRESENT, &err
	}

	// Ensure we clean up any stray connection

	var strbody []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		strbody, _ = ioutil.ReadAll(resp.Body)
	}

	if resp.StatusCode == 200 {
		return PRESENCE_PRESENT, nil
	}

	// else ...
	emsg := fmt.Errorf("Bad return status: (%d): %v",
		resp.StatusCode, string(strbody))
	return PRESENCE_NOT_PRESENT, &emsg
}

func queryNetworkStatus(ne NetEndpoint) (HSMEndpointPresence, *string, *error) {
	var res HSMEndpointPresence
	var errn *error

	if ne.name == "" {
		err := fmt.Errorf("endpoint name cannot be empty!")
		return PRESENCE_NOT_PRESENT, nil, &err
	}

	res, errn = queryNetworkStatusViaAddress(ne.name)
	if res == PRESENCE_PRESENT {
		return PRESENCE_PRESENT, &(ne.name), nil
	}

	rerr := fmt.Errorf("Not found. Tried %s: %s", ne.name, *errn)
	return PRESENCE_NOT_PRESENT, nil, &rerr
}

func watchForHardware(
	ne *NetEndpoint,
	quit chan struct{},
	netQuery func(NetEndpoint) (HSMEndpointPresence, *string, *error),
	onPresent func(NetEndpoint, string) *error,
	onNotPresent func(NetEndpoint) *error,
	loopLimit ...int) {

	var prevErr string
	var loopCount = 0

	log.Printf("INFO: Starting query thread for %s.  It is currently %s in HSM", ne.name,
		HSMEndpointPresenceToString[ne.HSMPresence])

	// Set the time for the fixed (minimum) wait between checkups
	// including a randomized wait at the start
	ticker := time.NewTicker(time.Duration(rand.Float32() * float32(startupVariableWaitMax) * float32(time.Second)))

	for {
		select {
		case <-ticker.C:

			// just to be safe, stop the ticker before we replace it...
			ticker.Stop()

			// Variable length wait at the start of checkup to make sure any bunching of checkup threads eventually shifts apart
			ticker = time.NewTicker(time.Duration((float32(checkupFixedWait) + (rand.Float32()-0.5)*2*float32(checkupVariableWaitMax)) * float32(time.Second)))

			go func() {
				ne.HSMPresLock.Lock()
				defer ne.HSMPresLock.Unlock()
				netPresence, addr, err := netQuery(*ne)
				if (err != nil) && (prevErr == "") {
					netPresence = ne.HSMPresence // ensure no state change on FIRST failure (but do one on second)
				}
				if err != nil {
					prevErr = fmt.Sprintf("%v", *err)
				} else {
					prevErr = ""
				}

				// Dont want to move items to present if there was an error reaching them.
				if netPresence == PRESENCE_PRESENT && ne.HSMPresence == PRESENCE_NOT_PRESENT && err == nil {
					err := (onPresent(*ne, *addr))
					if err != nil {
						log.Printf("WARNING: Failed to notify HSM that %s is now present: %v", ne.name, *err)
					} else {
						log.Printf("INFO: Marked %s ([%s]) present in HSM.", ne.name, *addr)
						ne.HSMPresence = PRESENCE_PRESENT
					}
				} else if netPresence == PRESENCE_NOT_PRESENT && ne.HSMPresence == PRESENCE_PRESENT {
					err := onNotPresent(*ne)
					if err != nil {
						log.Printf("WARNING: Failed to notify HSM that %s is NOT present: %v", ne.name, *err)
					} else {
						log.Printf("INFO: Lost network contact with %s", ne.name)
					}
				}
			}()

			if len(loopLimit) > 0 {
				if loopLimit[0] != 0 {
					loopCount++
					if loopCount >= loopLimit[0] {
						log.Printf("INFO: Quitting monitor thread for %s due to hitting loop count limit", ne.name)
						ticker.Stop()
						return
					}
				}
			}
		case <-quit:
			log.Printf("INFO: Quitting monitor thread for %s", ne.name)
			ticker.Stop()
			return
		}
	}
}

func watchForHSMChanges(quit chan struct{}) {
	log.Printf("INFO: Starting HSM query thread.")

	// Sync up with HSM every 5 mins
	ticker := time.NewTicker(300 * time.Second)
	defer ticker.Stop()

	// If we can't read from HSM try again every 30 seconds
	errTicker := time.NewTicker(30 * time.Second)
	defer errTicker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("TRACE: Checking up on HSM....")
			for {
				select {
				case <-errTicker.C:
					err := queryHSMState()
					if err == nil {
						break
					}
				case <-quit:
					return
				}
			}
		case <-quit:
			log.Printf("INFO: Quitting HSM monitor thread")
			return
		}
	}
}

// Do the dirty work of setting a parameter from an env var.

func __setenv_int(envval string, minval int, varp *int) {
	envstr := os.Getenv(envval)
	if envstr != "" {
		ival, err := strconv.Atoi(envstr)
		if err != nil {
			log.Println("ERROR converting env var", envval, ":", err,
				"-- setting unchanged.")
			return
		}
		*varp = ival
		if *varp < minval {
			*varp = minval
		}
	}
}

// Set parameters from env vars.

func getEnvVars() {
	var envstr string

	__setenv_int("MEDS_DEBUG", 0, &debugLevel)
	__setenv_int("MEDS_SMN_TIMEOUT", 1, &smnTimeoutSecs)

	envstr = os.Getenv("MEDS_NTP_TARG")
	if envstr != "" {
		ntpTarg = envstr
	}
	envstr = os.Getenv("MEDS_NTP_TARG_USE_IP")
	if envstr != "" {
		ntpTargUseIP = true
	}
	envstr = os.Getenv("MEDS_SYSLOG_TARG")
	if envstr != "" {
		syslogTarg = envstr
	}
	envstr = os.Getenv("MEDS_SYSLOG_TARG_USE_IP")
	if envstr != "" {
		syslogTargUseIP = true
	}
	envstr = os.Getenv("MEDS_NP_RF_URL")
	if envstr != "" {
		redfishNPSuffix = envstr
	}
	envstr = os.Getenv("MEDS_ROOT_USER")
	if envstr != "" {
		defUser = envstr
	}
	envstr = os.Getenv("MEDS_ROOT_PASSWORD")
	if envstr != "" {
		defPass = envstr
	}
	envstr = os.Getenv("MEDS_ROOT_SSH_KEY")
	if envstr != "" {
		defSSHKey = envstr
	}
	envstr = os.Getenv("MEDS_HSM")
	if envstr != "" {
		hsm = envstr
	}
	envstr = os.Getenv("MEDS_SLS")
	if envstr != "" {
		sls = envstr
	}
	envstr = os.Getenv("MEDS_CA_URI")
	if envstr != "" {
		hms_ca_uri = envstr
	}
	//These are for debugging/testing
	envstr = os.Getenv("MEDS_CA_PKI_URL")
	if envstr != "" {
		log.Printf("INFO: Using CA PKI URL: '%s'", envstr)
		hms_certs.ConfigParams.VaultCAUrl = envstr
	}
	envstr = os.Getenv("MEDS_VAULT_PKI_URL")
	if envstr != "" {
		log.Printf("INFO: Using VAULT PKI URL: '%s'", envstr)
		hms_certs.ConfigParams.VaultPKIUrl = envstr
	}
	envstr = os.Getenv("MEDS_VAULT_JWT_FILE")
	if envstr != "" {
		log.Printf("INFO: Using Vault JWT file: '%s'", envstr)
		hms_certs.ConfigParams.VaultJWTFile = envstr
	}
	envstr = os.Getenv("MEDS_K8S_AUTH_URL")
	if envstr != "" {
		log.Printf("INFO: Using K8S AUTH URL: '%s'", envstr)
		hms_certs.ConfigParams.K8SAuthUrl = envstr
	}
	envstr = os.Getenv("MEDS_LOG_INSECURE_FAILOVER")
	if envstr != "" {
		yn, _ := strconv.ParseBool(envstr)
		if yn == false {
			log.Printf("INFO: Not logging Redfish insecure failovers.")
			hms_certs.ConfigParams.LogInsecureFailover = false
		}
	}
	__setenv_int("MEDS_HTTP_TIMEOUT", 1, &clientTimeout)
}

func init_cabinet(cab GenericHardware) error {
	endpoints := make([]*NetEndpoint, 0)

	rackNum, rerr := strconv.Atoi(cab.Xname[1:])
	if rerr != nil {
		err := fmt.Errorf("INTERNAL ERROR, can't convert Xname '%s' to int: %v",
			cab.Xname, rerr)
		return err
	}

	var cabExtra sls_common.ComptypeCabinet
	ce, baerr := json.Marshal(cab.ExtraPropertiesRaw)
	if baerr != nil {
		err := fmt.Errorf("INTERNAL ERROR, can't marshal cab props: %v",
			baerr)
		log.Println(err)
		return err
	}

	baerr = json.Unmarshal(ce, &cabExtra)
	if baerr != nil {
		err := fmt.Errorf("INTERNAL ERROR, can't unmarshal cab props: %v",
			baerr)
		log.Println(err)
		return err
	}

	// Make sure the map checks out before reaching into it to avoid panic.
	hmnNetwork, networkExists := cabExtra.Networks["cn"]["HMN"]
	if !networkExists {
		err := fmt.Errorf("cabinet doesn't have HMN network for compute nodes: %+v\n", cabExtra)
		return err
	}

	macPrefix := ""
	if hmnNetwork.MACPrefix != "" {
		macPrefix = hmnNetwork.MACPrefix
	} else {
		// Default
		macPrefix = MAC_PREFIX
	}

	endpoints = append(endpoints, GenerateEnvironmentalControllerEndpoints(rackNum)...)
	endpoints = append(endpoints, GenerateChassisEndpoints(macPrefix, rackNum)...)

	// Determine what ethernet interfaces need to be get added or updated.
	hsmEthernetInterfaces, err := dhcpdnsClient.GetAllEthernetInterfaces()
	if err != nil {
		log.Println("Failed to get ethernet interfaces from HSM, not processing further: ", err)
		return err
	}
	hsmEthernetInterfacesMap := map[string]sm.CompEthInterfaceV2{}
	for _, ei := range hsmEthernetInterfaces {
		hsmEthernetInterfacesMap[ei.ID] = ei
	}

	// Generate MAC addresses for hardware in this cabinet
	for _, v := range endpoints {
		normalizedMAC := strings.ToLower(strings.ReplaceAll(v.mac, ":", ""))

		// Check to see if the generated endpoint has a MAC address associated with it.
		// Currently MEDS does generate a MAC addresses fro CEC's. Ex: x5000e0, x5000e1
		if normalizedMAC == "" {
			log.Printf("WARN: Endpoint has no MAC address: %s", v.name)
			continue
		}

		// Preload HSM EthernetInterfaces with the endpoints.
		ethernetInterface := sm.CompEthInterfaceV2{
			MACAddr: normalizedMAC,
			CompID:  v.name,
		}

		// POST/PATCH ethernet interfaces into HSM
		if hsmEI, ok := hsmEthernetInterfacesMap[ethernetInterface.MACAddr]; ok && hsmEI.CompID == ethernetInterface.CompID {
			// The MAC address is currently in HSM with the same component ID
			log.Printf("INFO: Ethernet interface for MAC %s and CompID %s already present in HSM", ethernetInterface.MACAddr, ethernetInterface.CompID)
		} else if ok && hsmEI.CompID != ethernetInterface.CompID {
			// The MAC address is currently in HSM with a different component ID
			log.Printf("INFO: Patching ethernet interface with MAC %s. HSM has CompID %s want %s.", ethernetInterface.MACAddr, hsmEI.CompID, ethernetInterface.CompID)
			patchErr := dhcpdnsClient.PatchEthernetInterface(ethernetInterface)

			if patchErr != nil {
				log.Println("ERROR: Failed to patch ethernet interface to HSM, not processing further: ", patchErr)
				log.Printf("Interface: %+v", ethernetInterface)

				// If the add to HSM fails don't add the endpoint to any lists and instead skip over it so we process it again.
				// The main loop will try to re-initialize the cabinet
				return patchErr
			}

			log.Printf("INFO: Patched new ethernet interface to HSM: %+v", ethernetInterface.CompID)
		} else {
			// Add the new ethernet interface. Patches instead if it's already present just in case
			addErr := dhcpdnsClient.AddNewEthernetInterface(ethernetInterface, true)

			if addErr != nil {
				log.Println("Failed to add new ethernet interface to HSM, not processing further: ", addErr)
				log.Printf("Interface: %+v", ethernetInterface)

				// If the add to HSM fails don't add the endpoint to any lists and instead skip over it so we process it again.
				// The main loop will try to re-initialize the cabinet
				return addErr
			}

			log.Printf("INFO: Added new ethernet interface to HSM: %+v", ethernetInterface.CompID)
		}
	}

	log.Printf("INFO: Finished adding EthernetInterfaces to HSM for cabinet %s", cab.Xname)

	// Verify that the FQDN/Hostname for RedfishEndpoints in HSM are what we expect
	verifyCabinetRedfishEndpoints(endpoints)

	// Start watching for hardware
	for _, v := range endpoints {
		// Create a channel we can use to kill this later
		v.QuitChannel = make(chan struct{})

		// Determine if this redfish endpoint is known in state manager
		hsmRedfishEndpointsCacheLock.Lock()
		if _, known := hsmRedfishEndpointsCache[v.name]; known {
			v.HSMPresence = PRESENCE_PRESENT
		}
		hsmRedfishEndpointsCacheLock.Unlock()

		// Now add endpoints to activeCabinets and
		activeCabinets[cab.Xname] = append(activeCabinets[cab.Xname], v)
		activeEndpoints[v.name] = v

		// Start hardware polling thread
		go watchForHardware(v, v.QuitChannel, queryNetworkStatus, notifyXnamePresent,
			notifyHSMXnameNotPresent)
	}

	return nil
}

func verifyCabinetRedfishEndpoints(endpoints []*NetEndpoint) error {
	// Verify that the FQDN/Hostname for RedfishEndpoints in HSM are what we expect
	for _, v := range endpoints {
		// Determine if this redfish endpoint is known in state manager and if it has the correct FQDN
		hsmRedfishEndpointsCacheLock.Lock()
		updateRFEndpoint := false
		var fqdn, hostname string
		if rfEP, known := hsmRedfishEndpointsCache[v.name]; known {

			// Verify that ChassisBMC's have the correct FQDN/hostname values set
			// TODO For authoritative DNS the following check should be changed to handle the FQDN of the system.
			if xnametypes.GetHMSType(rfEP.ID) == xnametypes.ChassisBMC && rfEP.ID != rfEP.FQDN {
				fqdn = rfEP.ID
				hostname = rfEP.ID
				log.Printf("Found ChassisBMC RedfishEndpoint with ID (%s) and FQDN (%s) PATCHING HSM to use FQDN (%s) and Hostname (%s)\n",
					v.name, rfEP.FQDN, fqdn, hostname)

				updateRFEndpoint = true
				rfEP.FQDN = fqdn
			}
		}
		hsmRedfishEndpointsCacheLock.Unlock()

		// Update the redfish endpoint if applicable
		if updateRFEndpoint {
			err := patchXnameFQDN(v.name, fqdn, hostname)
			if err != nil {
				log.Printf("Failed to update RedfishEndpoint (%s) in HSM with new FQDN/Hostname, not processing further: %v\n", v.name, err)

				// If the add to HSM fails don't add the endpoint to any lists and instead skip over it so we process it again.
				// The main loop will try to re-initialize the cabinet
				return err
			}
		}
	}

	return nil
}

func deinit_cab(k string) {
	// Iterate through the endpoints in the cabinet and stop them
	for endp := range activeCabinets[k] {
		log.Printf("TRACE: quitting %s", activeCabinets[k][endp].name)
		activeCabinets[k][endp].QuitChannel <- struct{}{}
		delete(activeEndpoints, activeCabinets[k][endp].name)
	}

	// Remove from active cabinets
	delete(activeCabinets, k)
}

// This function is used to set up an HTTP validated/non-validated client
// pair for Redfish operations.  This is done at the start of things, and also
// whenever the CA chain bundle is "rolled".

func setupRFHTTPStuff() error {
	var err error

	//Wait for all reader locks to release, prevent new reader locks.  Once
	//we acquire this lock, all RF operations are blocked until we unlock.

	rfClientLock.Lock()
	defer rfClientLock.Unlock()

	log.Printf("INFO: All RF threads paused.")
	if hms_ca_uri != "" {
		log.Printf("INFO: Creating Redfish TLS-secured client, CA URI: '%s'", hms_ca_uri)
	} else {
		log.Printf("INFO: Creating non-validated Redfish client, (no CA URI)")
	}

	rfClient, err = hms_certs.CreateHTTPClientPair(hms_ca_uri, clientTimeout)
	if err != nil {
		return fmt.Errorf("ERROR: Can't create TLS cert-enabled HTTP client: %v", err)
	}
	log.Printf("INFO: TLS-secured Redfish client successfully created.")

	var nwp bmc_nwprotocol.NWPData
	nwp.SyslogSpec = syslogTarg
	nwp.NTPSpec = ntpTarg
	nwp.CAChainURI = hms_ca_uri
	rfNWPStatic, err = bmc_nwprotocol.InitInstance(nwp, redfishNPSuffix, serviceName)
	if err != nil {
		return fmt.Errorf("ERROR setting up NW protocol handling: %v", err)
	}

	return nil
}

func caChangeCB(caBundle string) {
	log.Printf("INFO: CA bundle rolled; waiting for all RF threads to pause...")
	setupRFHTTPStuff()
	log.Printf("INFO: HTTP transports/clients now set up with new CA bundle.")
}

func main() {
	var credentialsVault string
	var err error

	printNodes = flag.Bool("print-nodes", false,
		"Print node records, otherwise print nC/sC/cC")
	flag.StringVar(&defUser, "default-username", "",
		"Default username to use when communicating with targets")
	flag.StringVar(&defPass, "default-password", "",
		"Default password to use when communicating with targets")
	flag.StringVar(&defSSHKey, "default-sshkey", "",
		"Default SSH key to use when communicating with targets")
	flag.StringVar(&sls, "sls", "http://cray-sls/v1",
		"Location of the System Layout Service API, up through the /v1 portion. (Do not include trailing slash)")
	flag.StringVar(&hsm, "hsm", "http://cray-smd/hsm/v2",
		"Location of the Hardware State Manager API, up through the /v2 portion. (Do not include trailing slash)")
	flag.StringVar(&syslogTarg, "syslog", "",
		"Server:Port of the syslog aggregator")
	flag.StringVar(&ntpTarg, "ntp", "",
		"Server:Port of the NTP service")
	flag.StringVar(&redfishNPSuffix, "np-rf-url", "/redfish/v1/Managers/BMC/NetworkProtocol",
		"URL path for network options Redfish endpoint")
	flag.StringVar(&credentialsVault, "credentialsVaultPrefix", model.CredentialsKeyPrefix,
		"Vault prefix for storing MEDS credentials")
	flag.IntVar(&maxInitialHSMSyncAttempts, "max-initial-hsm-sync-attempts", 30,
		"Number of attempts to perform an initial sync with HSM")
	flag.Parse()

	getEnvVars()

	serviceName, err = base.GetServiceInstanceName()
	if err != nil {
		log.Printf("Can't get service instance (hostname)!  Setting to 'MEDS'")
		serviceName = "MEDS"
	}
	log.Printf("Service Instance Name: '%s'", serviceName)

	log.Printf("Connecting to secure store (Vault)...")
	// Start a connection to Vault
	if ss, err := sstorage.NewVaultAdapter("secret"); err != nil {
		log.Printf("Error: Secure Store connection failed - %s", err)
		panic(err)
	} else {
		log.Printf("Connection to secure store (Vault) succeeded")
		credStorage = model.NewMedsCredStore(credentialsVault, ss)
		hcs = compcreds.NewCompCredStore("hms-creds", ss)
	}

	//Set up DNS/DHCP
	dhcpdnsClient = dns_dhcp.NewDHCPDNSHelperInstance(hsm, nil, serviceName)

	//Set up RF HTTP client and NWP stuff

	hms_certs.InitInstance(nil, serviceName)

	log.Printf("INFO: Setting up non-TLS-validated HTTP client for in-service use.")
	client, _ = hms_certs.CreateHTTPClientPair("", clientTimeout)

	//Fix up syslog/NTP IP/hostnames

	var nwp bmc_nwprotocol.NWPData

	//Check if we are to use IP addresses, and if so, convert them here.
	if syslogTargUseIP {
		toks := strings.Split(syslogTarg, ":")
		ip, iperr := net.LookupIP(toks[0])
		if iperr != nil {
			log.Printf("ERROR looking up syslog server IP addr: %v", iperr)
			log.Printf("Using hostname anyway.")
		} else {
			syslogTarg = ip[0].String()
			if len(toks) > 1 {
				syslogTarg = syslogTarg + ":" + toks[1]
			} else {
				log.Printf("INFO: No port specified in syslog target, using 123.")
				syslogTarg = syslogTarg + ":123"
			}
		}
	}
	if ntpTargUseIP {
		toks := strings.Split(ntpTarg, ":")
		ip, iperr := net.LookupIP(toks[0])
		if iperr != nil {
			log.Printf("ERROR looking up NTP IP addr: %v", iperr)
			log.Printf("Using hostname anyway.")
		} else {
			ntpTarg = ip[0].String()
			if len(toks) > 1 {
				ntpTarg = ntpTarg + ":" + toks[1]
			} else {
				log.Printf("INFO: No port specified in NTP target, using 514.")
				ntpTarg = ntpTarg + ":514"
			}
		}
	}

	nwp.SyslogSpec = syslogTarg
	nwp.NTPSpec = ntpTarg
	log.Printf("Using syslog server: '%s'", syslogTarg)
	log.Printf("Using NTP server: '%s'", ntpTarg)

	rfNWPStatic, err = bmc_nwprotocol.InitInstance(nwp, redfishNPSuffix, serviceName)
	if err != nil {
		log.Println("ERROR setting up NW protocol handling:", err)
		//TODO: should we exit??
	}

	//Set up RF HTTP transport.  Re-try for Vault, fail over on too many retries.

	ok := false
	for ix := 1; ix <= 10; ix++ {
		err := setupRFHTTPStuff()
		if err == nil {
			log.Printf("INFO: Successfully set up Redfish transport.")
			ok = true
			break
		}
		log.Printf("ERROR: RF transport attempt %d: %v", ix, err)
		time.Sleep(3 * time.Second)
	}

	if !ok {
		log.Printf("ERROR: exhausted all retries creating TLS-secured Redfish transport, failing over insecure.")
		hms_ca_uri = ""
		err = setupRFHTTPStuff()
		if err != nil {
			panic("ERROR: can't create any RF HTTP transport!!!!!")
		}
	}

	if hms_ca_uri != "" {
		err = hms_certs.CAUpdateRegister(hms_ca_uri, caChangeCB)
		if err != nil {
			log.Printf("WARNING: Unable to register CA bundle watcher for URI: '%s': %v",
				hms_ca_uri, err)
			log.Printf("   This means no updates when CA bundle is rolled.")
		} else {
			log.Printf("INFO: Registered CA bundle watcher for URI: '%s'", hms_ca_uri)
		}
	} else {
		log.Printf("WARNING: No CA bundle URI specified, not watching for CA changes.")
	}

	/* Start up watch for HSM changes early, so we can loop over data */
	HSMPollquitc := make(chan struct{})

	// Perform an initial sync with HSM before starting, so we now the state of the redfish endpoints
	// This will help prevent MEDS from flooding HSM with discoveries when it starts up
	for attempt := 1; attempt <= maxInitialHSMSyncAttempts; attempt++ {
		err := queryHSMState()
		if err != nil {
			log.Printf("Initial sync with HSM failed. attempt %d of %d", attempt, maxInitialHSMSyncAttempts)
			time.Sleep(time.Second)
		} else {
			log.Printf("Successfully performed initial sync with HSM")
			break
		}

		if attempt >= maxInitialHSMSyncAttempts {
			log.Fatal("Unable to perform initial sync with HSM after reaching max attempts")
		}
	}

	// TODO I'll have to rewrite how this is handled, I think.  Or at least move the function into the thread
	go watchForHSMChanges(HSMPollquitc)

	// With SLS enabled we want to update ourselves periodically.
	basetime := 30 * time.Second
	backoffTime := 5 * time.Second
	maxtime := 5 * time.Minute
	waittime := basetime
	for {
		log.Printf("INFO: Sleeping %d seconds before refreshing data", waittime/time.Second)
		time.Sleep(waittime)

		cabinets, err := getSLSCabInfo()
		if err != nil {
			log.Printf("WARNING: Can't get cabinet list from SLS: %v\n",
				err)
			waittime += backoffTime
			if waittime > maxtime {
				waittime = maxtime
			}
			continue
		}
		if len(cabinets) == 0 {
			log.Printf("INFO: No cabinets found in SLS.\n")
		}

		// List of cabinets. We'll remove those we find in SLS from this
		oldCabList := make(map[string]bool, 0)
		for k := range activeCabinets {
			oldCabList[k] = true
		}

		rfClientLock.RLock()
		activeEndpointsLock.Lock() // Take the lock so we can update!
		for _, cab := range cabinets {
			log.Printf("TRACE: Handling cabinet %s from SLS", cab.Xname)
			if _, ok := activeCabinets[cab.Xname]; !ok {
				log.Printf("TRACE: Cabinet %s is new", cab.Xname)
				// Cabinet not present, need to set up and init everything
				err := init_cabinet(cab)
				if err != nil {
					log.Printf("Error initializing cabinet: %s", err)
					continue
				}
			} else {
				// Else this cabinet is already present
				// Take no action
				log.Printf("TRACE: Cabinet %s is not new", cab.Xname)
			}

			// No matter hat though, we need to remove it from oldCabList to account for finding it
			delete(oldCabList, cab.Xname)

		}

		// Anything left in oldCabList disappeared.
		for k := range oldCabList {
			deinit_cab(k)
		}

		activeEndpointsLock.Unlock()
		rfClientLock.RUnlock()

		waittime = basetime
	}
}
