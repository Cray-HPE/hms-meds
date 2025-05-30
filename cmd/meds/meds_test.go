/*
 * MIT License
 *
 * (C) Copyright [2019-2022,2025] Hewlett Packard Enterprise Development LP
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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	base "github.com/Cray-HPE/hms-base/v2"
	"github.com/Cray-HPE/hms-certs/pkg/hms_certs"
)

func NetEndpointEquals(a NetEndpoint, b NetEndpoint) bool {
	if a.name != b.name {
		print("Names not equal")
		return false
	} else if a.mac != b.mac {
		print("MACs not equal")
		return false
	} else if a.hwtype != b.hwtype {
		print("hwtypess not equal")
		return false
	}
	return true
}

func Test_GenerateMAC(t *testing.T) {
	ret := GenerateMAC("02", 7, 5, 3, 1)
	ec := "02:00:07:05:03:10"
	if ret != ec {
		t.Errorf("Generated MAC did not match expectation:\nExpectation:\n%v\nGot:\n%v\n", ec, ret)
	}
}

func Test_GenerateMACnC(t *testing.T) {
	ret := GenerateMACnC("02", 7, 5, 3, 1)
	ec := "02:00:07:05:33:10"
	if ret != ec {
		t.Errorf("Generated nC MAC did not match expectation:\nExpectation:\n%v\nGot:\n%v\n", ec, ret)
	}
}

func Test_GenerateMACsC(t *testing.T) {
	ret := GenerateMACsC("02", 7, 5, 3)
	ec := "02:00:07:05:63:00"
	if ret != ec {
		t.Errorf("Generated sC MAC did not match expectation:\nExpectation:\n%v\nGot:\n%v\n", ec, ret)
	}
}

func Test_GenerateMACcC(t *testing.T) {
	ret := GenerateMACcC("02", 7, 5)
	ec := "02:00:07:05:00:00"
	if ret != ec {
		t.Errorf("Generated cC MAC did not match expectation:\nExpectation:\n%v\nGot:\n%v\n", ec, ret)
	}
}

func Test_GenerateEnvironmentalControllerEndpoints(t *testing.T) {
	ret := GenerateEnvironmentalControllerEndpoints(3)

	ec1 := NetEndpoint{
		name:   "x3e0",
		mac:    "",
		hwtype: TYPE_ENV_CONTROLLER,
	}
	ec2 := NetEndpoint{
		name:   "x3e1",
		mac:    "",
		hwtype: TYPE_ENV_CONTROLLER,
	}

	if !NetEndpointEquals(*(ret[0]), ec1) {
		t.Errorf("First eC did not match exectation:\nExpectation:\n%v\nGot:\n%v\n", ec1, *(ret[0]))
	}
	if !NetEndpointEquals(*(ret[1]), ec2) {
		t.Errorf("Second eC did not match exectation:\nExpectation:\n%v\nGot:\n%v\n", ec2, *(ret[1]))
	}
}

func Test_GenerateNodeCardEndpoints(t *testing.T) {
	ret := GenerateNodeCardEndpoints("02", 7, 5, 3)

	nc2 := NetEndpoint{
		name:   "x7c5s3b1",
		mac:    "02:00:07:05:33:10",
		hwtype: TYPE_NODE_CARD,
	}

	if !NetEndpointEquals(*(ret[1]), nc2) {
		t.Errorf("nC did not match exectation:\nExpectation:\n%v\nGot:\n%v\n", nc2, *(ret[1]))
	}
}

func Test_GenerateSwitchCardEndpoints(t *testing.T) {
	ret := GenerateSwitchCardEndpoints("02", 7, 5)

	sc1 := NetEndpoint{
		name:   "x7c5r0b0",
		mac:    "02:00:07:05:60:00",
		hwtype: TYPE_SWITCH_CARD,
	}

	if !NetEndpointEquals(*(ret[0]), sc1) {
		t.Errorf("Switch card did not match exectation:\nExpectation:\n%v\nGot:\n%v\n", sc1, *(ret[0]))
	}

	if len(ret) != MTN_SWITCH_COUNT*(1+MTN_nC_PER_SLOT) {
		t.Errorf("Endpoints is the wrong length.  Is: %d, should be %d", len(ret), MTN_SWITCH_COUNT*(1+MTN_nC_PER_SLOT))
	}
}

func Test_GenerateChassisEndpoints(t *testing.T) {
	ret := GenerateChassisEndpoints("02", 7, []int{0, 1, 2, 3, 4, 5, 6, 7})

	cha1 := NetEndpoint{
		name:   "x7c0b0",
		mac:    "02:00:07:00:00:00",
		hwtype: TYPE_CHASSIS,
	}

	if !NetEndpointEquals(*(ret[0]), cha1) {
		t.Errorf("Chassis did not match exectation:\nExpectation:\n%v\nGot:\n%v\n", cha1, *(ret[0]))
	}

	if len(ret) != MTN_CHASSIS_COUNT*(1+MTN_SWITCH_COUNT*(1+MTN_nC_PER_SLOT)) {
		t.Errorf("Endpoints is the wrong length.  Is: %d, should be %d", len(ret), MTN_CHASSIS_COUNT*(1+MTN_SWITCH_COUNT*(1+MTN_nC_PER_SLOT)))
	}
}

var queryHSM_response HSMEndpointPresence
var queryHSM_error error
var queryHSM_count int

func configure_queryHSM(resp HSMEndpointPresence, eresp error) {
	queryHSM_response = resp
	queryHSM_error = eresp
	queryHSM_count = 0
}

func mock_queryHSM(eps []*NetEndpoint) error {
	queryHSM_count += 1
	for _, ep := range eps {
		ep.HSMPresence = queryHSM_response
	}
	return queryHSM_error
}

func mock_queryHSM_1stError(eps []*NetEndpoint) error {
	queryHSM_count += 1
	if queryHSM_count <= 1 {
		err := errors.New("Dummy!")
		for _, ep := range eps {
			ep.HSMPresence = PRESENCE_NOT_PRESENT
		}
		return err
	}
	for _, ep := range eps {
		ep.HSMPresence = queryHSM_response
	}
	return queryHSM_error
}

var queryNet_response HSMEndpointPresence
var queryNet_respAddr *string
var queryNet_error *error
var queryNet_count int

func configure_queryNet(resp HSMEndpointPresence, respAddr *string, eresp *error) {
	queryNet_response = resp
	queryNet_respAddr = respAddr
	queryNet_error = eresp
	queryNet_count = 0
}

func mock_queryNet(ne NetEndpoint) (HSMEndpointPresence, *string, *error) {
	queryNet_count += 1
	return queryNet_response, queryNet_respAddr, queryNet_error
}

var notifyHSMPresentCalls []NetEndpoint
var notifyHSMPresentResponse *error

func configure_notifyHSMPresent(err *error) {
	notifyHSMPresentResponse = err
	notifyHSMPresentCalls = make([]NetEndpoint, 0)
}

func mock_notifyHSMPresent(xname NetEndpoint, addr string) *error {
	notifyHSMPresentCalls = append(notifyHSMPresentCalls, xname)
	return notifyHSMPresentResponse
}

var notifyHSMNotPresentCalls []NetEndpoint
var notifyHSMNotPresentResponse *error

func configure_notifyHSMNotPresent(err *error) {
	notifyHSMNotPresentResponse = err
	notifyHSMNotPresentCalls = make([]NetEndpoint, 0)
}

func mock_notifyHSMNotPresent(xname NetEndpoint) *error {
	notifyHSMNotPresentCalls = append(notifyHSMNotPresentCalls, xname)
	return notifyHSMNotPresentResponse
}

func Test_watchForHardware_notPresentToPresent(t *testing.T) {

	// Some of the tests depend on the length of waits in checkup threads.  This sets them really short:
	checkupVariableWaitMax = 0
	checkupFixedWait = 1
	startupVariableWaitMax = 1

	if testing.Short() {
		t.Skip("Skipping test_watchForHardware_notPresentToPresent as we're only running short tests")
	}
	quit := make(chan struct{})
	node := NetEndpoint{
		name:        "testNode",
		mac:         "001cedc0ffee",
		hwtype:      TYPE_CHASSIS,
		HSMPresence: PRESENCE_NOT_PRESENT,
	}
	configure_queryNet(PRESENCE_PRESENT, &(node.name), nil)
	configure_notifyHSMPresent(nil)
	configure_notifyHSMNotPresent(nil)

	watchForHardware(
		&node, quit, mock_queryNet, mock_notifyHSMPresent,
		mock_notifyHSMNotPresent, 2)

	close(quit)

	// Because the mock functions are called in goroutines, we have to wait for them to complete
	time.Sleep(1 * time.Second)

	// Check calls to mock_notifyHSMPresent
	if len(notifyHSMPresentCalls) != 1 {
		t.Errorf("Wrong number of calls to mock_notifyHSMPresent.  Expected 1, got %d",
			len(notifyHSMPresentCalls))
	}

	if notifyHSMPresentCalls[0].name != node.name {
		t.Errorf("Unexpected data passed to mock_notifyHSMPresent.  Got\n%v\nExpected\n%v",
			node, notifyHSMPresentCalls[0])
	}

	if len(notifyHSMNotPresentCalls) != 0 {
		t.Errorf("Wrong number of calls to mock_notifyHSMNotPresent.  Expected 1, got %d",
			len(notifyHSMNotPresentCalls))
	}

	if queryNet_count != 2 {
		t.Errorf("Wrong number of calls to queryNet.  Expected 2, got %d", queryNet_count)
	}

	// Verify the NetEndpoint has the correct HSMPresence state
	if node.HSMPresence != PRESENCE_PRESENT {
		t.Errorf("Unexpected HSM Presence state: %d", node.HSMPresence)
	}
}

func Test_watchForHardware_presentToNotPresent(t *testing.T) {
	// Some of the tests depend on the length of waits in checkup threads.  This sets them really short:
	checkupVariableWaitMax = 0
	checkupFixedWait = 5
	startupVariableWaitMax = 1

	if testing.Short() {
		t.Skip("Skipping test_watchForHardware_presentToNotPresent as we're only running short tests")
	}
	quit := make(chan struct{})
	node := NetEndpoint{
		name:        "testNode",
		mac:         "001cedc0ffee",
		hwtype:      TYPE_CHASSIS,
		HSMPresence: PRESENCE_PRESENT,
	}
	err := errors.New("Dummy: Can't find endpoint")
	configure_queryNet(PRESENCE_NOT_PRESENT, nil, &err)
	configure_notifyHSMPresent(nil)
	configure_notifyHSMNotPresent(nil)

	watchForHardware(
		&node, quit, mock_queryNet, mock_notifyHSMPresent,
		mock_notifyHSMNotPresent, 2)

	close(quit)

	// Because the mock functions are called in goroutines, we have to wait for them to complete
	time.Sleep(1 * time.Second)

	// Check calls to mock_notifyHSMPresent
	if len(notifyHSMPresentCalls) != 0 {
		t.Errorf("Wrong number of calls to mock_notifyHSMPresent.  Expected 0, got %d",
			len(notifyHSMPresentCalls))
	}

	if len(notifyHSMNotPresentCalls) != 1 {
		t.Errorf("Wrong number of calls to mock_notifyHSMNotPresent.  Expected 1, got %d",
			len(notifyHSMNotPresentCalls))
	}

	if notifyHSMNotPresentCalls[0].name != node.name {
		t.Errorf("Unexpected data passed to mock_notifyHSMNotPresent.  Got\n%v\nExpected\n%v",
			node, notifyHSMNotPresentCalls[0])
	}

	if queryNet_count != 2 {
		t.Errorf("Wrong numbber of calls to queryNet.  Expected 2, got %d", queryNet_count)
	}

	// Verify the NetEndpoint has the correct HSMPresence state
	// Yes, we really expect the node to be PRESENT in HSM when BMC is not reachable
	if node.HSMPresence != PRESENCE_PRESENT {
		t.Errorf("Unexpected HSM Presence state: %d", node.HSMPresence)
	}
}

func Test_watchForHardware_noChange_0(t *testing.T) {
	// Some of the tests depend on the length of waits in checkup threads.  This sets them really short:
	checkupVariableWaitMax = 0
	checkupFixedWait = 1
	startupVariableWaitMax = 1

	if testing.Short() {
		t.Skip("Skipping test_watchForHardware_noChange_0 as we're only running short tests")
	}
	quit := make(chan struct{})
	node := NetEndpoint{
		name:        "testNode",
		mac:         "001cedc0ffee",
		hwtype:      TYPE_CHASSIS,
		HSMPresence: PRESENCE_PRESENT,
	}
	configure_queryNet(PRESENCE_PRESENT, &(node.name), nil)
	configure_notifyHSMPresent(nil)
	configure_notifyHSMNotPresent(nil)

	watchForHardware(
		&node, quit, mock_queryNet, mock_notifyHSMPresent,
		mock_notifyHSMNotPresent, 2)

	close(quit)

	// Because the mock functions are called in goroutines, we have to wait for them to complete
	time.Sleep(1 * time.Second)

	// Check calls to mock_notifyHSMPresent
	if len(notifyHSMPresentCalls) != 0 {
		t.Errorf("Wrong number of calls to mock_notifyHSMPresent.  Expected 0, got %d",
			len(notifyHSMPresentCalls))
	}

	if len(notifyHSMNotPresentCalls) != 0 {
		t.Errorf("Wrong number of calls to mock_notifyHSMNotPresent.  Expected 0, got %d",
			len(notifyHSMNotPresentCalls))
	}

	if queryNet_count != 2 {
		t.Errorf("Wrong numbber of calls to queryNet.  Expected 2, got %d", queryNet_count)
	}

	// Verify the NetEndpoint has the correct HSMPresence state
	if node.HSMPresence != PRESENCE_PRESENT {
		t.Errorf("Unexpected HSM Presence state: %d", node.HSMPresence)
	}
}

func Test_watchForHardware_noChange_1(t *testing.T) {
	// Some of the tests depend on the length of waits in checkup threads.  This sets them really short:
	checkupVariableWaitMax = 0
	checkupFixedWait = 1
	startupVariableWaitMax = 1

	if testing.Short() {
		t.Skip("Skipping test_watchForHardware_noChange_1 as we're only running short tests")
	}
	quit := make(chan struct{})
	node := NetEndpoint{
		name:        "testNode",
		mac:         "001cedc0ffee",
		hwtype:      TYPE_CHASSIS,
		HSMPresence: PRESENCE_NOT_PRESENT,
	}
	err := errors.New("Dummy: Can't find endpoint")
	configure_queryNet(PRESENCE_NOT_PRESENT, nil, &err)
	configure_notifyHSMPresent(nil)
	configure_notifyHSMNotPresent(nil)

	watchForHardware(
		&node, quit, mock_queryNet, mock_notifyHSMPresent,
		mock_notifyHSMNotPresent, 2)

	close(quit)

	// Because the mock functions are called in goroutines, we have to wait for them to complete
	time.Sleep(1 * time.Second)

	// Check calls to mock_notifyHSMPresent
	if len(notifyHSMPresentCalls) != 0 {
		t.Errorf("Wrong number of calls to mock_notifyHSMPresent.  Expected 0, got %d",
			len(notifyHSMPresentCalls))
	}

	if len(notifyHSMNotPresentCalls) != 0 {
		t.Errorf("Wrong number of calls to mock_notifyHSMNotPresent.  Expected 0, got %d",
			len(notifyHSMNotPresentCalls))
	}

	if queryNet_count != 2 {
		t.Errorf("Wrong numbber of calls to queryNet.  Expected 2, got %d", queryNet_count)
	}

	// Verify the NetEndpoint has the correct HSMPresence state
	if node.HSMPresence != PRESENCE_NOT_PRESENT {
		t.Errorf("Unexpected HSM Presence state: %d", node.HSMPresence)
	}
}

// This test verifies that no matter what netQuery returns,
// watchForHardware won't do a state change.  It set this up by making
// the node "not present" in HSM, then returning both a PRESENCE_PRESENT
// and an error in netQuery
func Test_watchForHardware_netQuery_FailureRecovery(t *testing.T) {
	// Some of the tests depend on the length of waits in checkup threads.  This sets them really short:
	checkupVariableWaitMax = 0
	checkupFixedWait = 1
	startupVariableWaitMax = 1

	if testing.Short() {
		t.Skip("Skipping Test_watchForHardware_HSMFailureRecovery as we're only running short tests")
	}
	quit := make(chan struct{})
	node := NetEndpoint{
		name:        "testNode",
		mac:         "001cedc0ffee",
		hwtype:      TYPE_CHASSIS,
		HSMPresence: PRESENCE_NOT_PRESENT,
	}
	err := errors.New("dummy")

	configure_queryNet(PRESENCE_PRESENT, nil, &err)
	configure_notifyHSMPresent(nil)
	configure_notifyHSMNotPresent(nil)

	watchForHardware(
		&node, quit, mock_queryNet, mock_notifyHSMPresent,
		mock_notifyHSMNotPresent, 2)

	time.Sleep(1 * time.Second)

	close(quit)

	if len(notifyHSMPresentCalls) != 0 {
		t.Errorf("Wrong number of calls to mock_notifyHSMPresent.  Expected 0, got %d",
			len(notifyHSMPresentCalls))
	}

	if len(notifyHSMNotPresentCalls) != 0 {
		t.Errorf("Wrong number of calls to mock_notifyHSMNotPresent.  Expected 0, got %d",
			len(notifyHSMNotPresentCalls))
	}

	// Verify the NetEndpoint has the correct HSMPresence state
	if node.HSMPresence != PRESENCE_NOT_PRESENT {
		t.Errorf("Unexpected HSM Presence state: %d", node.HSMPresence)
	}
}

func userAgentHeaderPresent(r *http.Request) bool {
	vlist, ok := r.Header[base.USERAGENT]
	if ok {
		for _, v := range vlist {
			if v == serviceName {
				return true
			}
		}
	}
	return false
}

func Test_notifyHSMXnamePresent(t *testing.T) {
	type HTTPResponse struct {
		respCode        int
		respBody        string
		expectedReqBody []byte
	}

	tests := []struct {
		description string
		responses   map[string]HTTPResponse
		nodeIn      NetEndpoint
		expectErr   bool
	}{{
		"Success (201) from HSM",
		map[string]HTTPResponse{
			"/Inventory/RedfishEndpoints": HTTPResponse{
				201,
				`[{"URI": "/hsm/v2/Inventory/RedfishEndpoints/x0c0s0b0"}]`,
				json.RawMessage(`{"ID":"x7c5s3b1","FQDN":"x7c5s3b1","MACAddr":"02:00:07:05:33:10","RediscoverOnUpdate":true}`),
			},
		},
		NetEndpoint{
			name:   "x7c5s3b1",
			mac:    "02:00:07:05:33:10",
			hwtype: TYPE_NODE_CARD,
		},
		false,
	}, {
		"Error (400) from HSM",
		map[string]HTTPResponse{
			"/Inventory/RedfishEndpoints": HTTPResponse{
				400,
				`{"type":"about:blank","detail":"Detail about this specific problem occurrence. See RFC7807","instance":"","status":400,"title":"Description of HTTP Status code, e.g. 400"}`,
				json.RawMessage(`{"ID":"x7c5s3b1","FQDN":"x7c5s3b1","MACAddr":"02:00:07:05:33:10","RediscoverOnUpdate":true}`),
			},
		},
		NetEndpoint{
			name:   "x7c5s3b1",
			mac:    "02:00:07:05:33:10",
			hwtype: TYPE_NODE_CARD,
		},
		true,
	}, {
		"Patch instead of fail on existing node",
		map[string]HTTPResponse{
			"/Inventory/RedfishEndpoints": HTTPResponse{
				409,
				`{"type":"about:blank","title":"Conflict","detail":"operation would conflict with an existing resource that has the same FQDN or xname ID.","status":409}`,
				json.RawMessage(`{"ID":"x7c5s3b1","FQDN":"x7c5s3b1","MACAddr":"02:00:07:05:33:10","RediscoverOnUpdate":true}`),
			},
			"/Inventory/RedfishEndpoints/x7c5s3b1": HTTPResponse{
				200,
				``,
				json.RawMessage(`{"ID":"x7c5s3b1","RediscoverOnUpdate":true,"Enabled":true}`),
			},
		},
		NetEndpoint{
			name:   "x7c5s3b1",
			mac:    "02:00:07:05:33:10",
			hwtype: TYPE_NODE_CARD,
		},
		false,
	}, {
		"Patch instead of fail on existing node (patch 404s)",
		map[string]HTTPResponse{
			"/Inventory/RedfishEndpoints": HTTPResponse{
				409,
				`{"type":"about:blank","title":"Conflict","detail":"operation would conflict with an existing resource that has the same FQDN or xname ID.","status":409}`,
				json.RawMessage(`{"ID":"x7c5s3b1","FQDN":"x7c5s3b1","MACAddr":"02:00:07:05:33:10","RediscoverOnUpdate":true}`),
			},
			"/Inventory/RedfishEndpoints/x7c5s3b1": HTTPResponse{
				404,
				`{"type":"about:blank","detail":"Not found","instance":"","status":404,"title":"Not found"}`,
				json.RawMessage(`{"ID":"x7c5s3b1","RediscoverOnUpdate":true,"Enabled":true}`),
			},
		},
		NetEndpoint{
			name:   "x7c5s3b1",
			mac:    "02:00:07:05:33:10",
			hwtype: TYPE_NODE_CARD,
		},
		true,
	}, {
		"Patch instead of fail on existing node (patch 400s)",
		map[string]HTTPResponse{
			"/Inventory/RedfishEndpoints": HTTPResponse{
				409,
				`{"type":"about:blank","title":"Conflict","detail":"operation would conflict with an existing resource that has the same FQDN or xname ID.","status":409}`,
				json.RawMessage(`{"ID":"x7c5s3b1","FQDN":"x7c5s3b1","MACAddr":"02:00:07:05:33:10","RediscoverOnUpdate":true}`),
			},
			"/Inventory/RedfishEndpoints/x7c5s3b1": HTTPResponse{
				400,
				`{"type":"about:blank","detail":"bad input","instance":"","status":400,"title":"bad input"}`,
				json.RawMessage(`{"ID":"x7c5s3b1","RediscoverOnUpdate":true,"Enabled":true}`),
			},
		},
		NetEndpoint{
			name:   "x7c5s3b1",
			mac:    "02:00:07:05:33:10",
			hwtype: TYPE_NODE_CARD,
		},
		true,
	}}

	serviceName = "MEDS_TEST"
	defUser = "root"
	defPass = "********"
	client, _ = hms_certs.CreateHTTPClientPair("", clientTimeout)
	setupRFHTTPStuff()

	for i, test := range tests {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath := r.URL.Path
			requestBody, _ := ioutil.ReadAll(r.Body)

			httpr, ok := test.responses[requestPath]
			if ok {
				// Check for User-Agent headers
				if !userAgentHeaderPresent(r) {
					t.Errorf("Test %v had no User-Agent header.", i)
				}

				// Check the request is the one we expected
				if bytes.Compare(httpr.expectedReqBody, requestBody) != 0 {
					t.Errorf("Test %v (%s) Failed: Expected request body is '%v'; Received '%v'", i, test.description, string(httpr.expectedReqBody), string(requestBody))
				}

				w.WriteHeader(httpr.respCode)
				w.Write(json.RawMessage(httpr.respBody))
			} else {
				w.WriteHeader(500)
				w.Write([]byte("Couldn't find HTTPResponse to give for URL " + requestPath))
			}
		}))
		defer testServer.Close()
		hsm = testServer.URL

		err := (notifyHSMXnamePresent(test.nodeIn, "10.0.0.1"))
		if !test.expectErr {
			if err != nil {
				t.Errorf("Test %v (%s) Failed: Received unexpected error - %v", i, test.description, err)
			}
		} else if err == nil {
			t.Errorf("Test %v (%s) Failed: Expected an error", i, test.description)
		}
	}
}

func Test_queryHSMState(t *testing.T) {
	tests := []struct {
		description      string
		respCode         int
		epsIn            map[string]*NetEndpoint
		expectedReqURI   string
		expectedPresence HSMEndpointPresence
		expectErr        bool
	}{{
		"Success (200) from HSM.",
		200,
		map[string]*NetEndpoint{
			"x7c5s3b1": &NetEndpoint{name: "x7c5s3b1"},
		},
		"/Inventory/RedfishEndpoints",
		PRESENCE_PRESENT,
		false,
	}, {
		"Not found (404) from HSM.",
		404,
		map[string]*NetEndpoint{
			"x7c5s3b1": &NetEndpoint{name: "x7c5s3b1"},
		},
		"/Inventory/RedfishEndpoints",
		PRESENCE_NOT_PRESENT,
		false,
	}, {
		"Error (500) from HSM.",
		500,
		map[string]*NetEndpoint{
			"x7c5s3b1": &NetEndpoint{name: "x7c5s3b1"},
		},
		"/Inventory/RedfishEndpoints",
		PRESENCE_NOT_PRESENT,
		true,
	}}
	var responseCode int
	var requestURI string
	serviceName = "MEDS_TEST"
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.URL.String()
		if responseCode == 200 {
			// Check for User-Agent headers
			if !userAgentHeaderPresent(r) {
				t.Errorf("Request %s had no User-Agent header.", requestURI)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(responseCode)
			w.Write(json.RawMessage(`{"RedfishEndpoints":[{"ID":"x7c5s3b1","Type":"Node"}]}`))
		} else if responseCode == 404 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write(json.RawMessage(`{"RedfishEndpoints":[]}`))
		} else {
			w.WriteHeader(responseCode)
			w.Write(json.RawMessage(`{"type":"about:blank","detail":"Detail about this specific problem occurrence. See RFC7807","instance":"","status":500,"title":"Description of HTTP Status code, e.g. 500"}`))
		}
	}))
	defer testServer.Close()
	hsm = testServer.URL

	for i, test := range tests {
		responseCode = test.respCode
		requestURI = ""
		activeEndpoints = test.epsIn
		err := (queryHSMState())
		if !test.expectErr {
			if err != nil {
				t.Errorf("Test %v (%s) Failed: Received unexpected error - %v", i, test.description, err)
			} else {
				if test.expectedReqURI != requestURI {
					t.Errorf("Test %v (%s) Failed: Expected request URI is '%v'; Received '%v'", i, test.description, test.expectedReqURI, requestURI)
				}
				if test.epsIn["x7c5s3b1"].HSMPresence != test.expectedPresence {
					t.Errorf("Test %v (%s) Failed: Expected component presence is '%v'; Received '%v'", i, test.description, HSMEndpointPresenceToString[test.expectedPresence], HSMEndpointPresenceToString[test.epsIn["x7c5s3b1"].HSMPresence])
				}
			}
		} else if err == nil {
			t.Errorf("Test %v (%s) Failed: Expected an error", i, test.description)
		}
	}
}

func Test_queryNetworkStatus(t *testing.T) {
	tests := []struct {
		description      string
		respCode         int
		expectedReqURI   string
		expectedPresence HSMEndpointPresence
		expectErr        bool
	}{{
		"Success (200). Present",
		200,
		"/redfish/v1/",
		PRESENCE_PRESENT,
		false,
	}, {
		"Error (400). Not present",
		400,
		"/redfish/v1/",
		PRESENCE_NOT_PRESENT,
		true,
	}}
	var responseCode int
	var requestURI string
	serviceName = "MEDS_TEST"
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.URL.String()
		if responseCode == 200 {
			// Check for User-Agent headers
			if !userAgentHeaderPresent(r) {
				t.Errorf("Request %s had no User-Agent header.", requestURI)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(responseCode)
			// The response payload doesn't really matter since queryNetworkStatus() doesn't parse it.
			w.Write(json.RawMessage(`{"ID":"x0c0s0b0n0","Type":"Node"}`))
		} else {
			w.WriteHeader(responseCode)
			w.Write(json.RawMessage(`{"type":"about:blank","detail":"Detail about this specific problem occurrence. See RFC7807","instance":"","status":500,"title":"Description of HTTP Status code, e.g. 500"}`))
		}
	}))
	defer testServer.Close()
	strs := strings.Split(testServer.URL, "//")
	address := strs[1]

	fmt.Printf("Server address is: %s", address)

	endpoint := NetEndpoint{
		name:   address,
		mac:    "02:00:07:05:33:10",
		hwtype: TYPE_NODE_CARD,
	}

	for i, test := range tests {
		responseCode = test.respCode
		requestURI = ""
		isPresent, _, err := queryNetworkStatus(endpoint)
		if isPresent != test.expectedPresence {
			t.Errorf("Test %v (%s) Failed: Expected component presence is '%v'; Received '%v'", i, test.description, HSMEndpointPresenceToString[test.expectedPresence], HSMEndpointPresenceToString[isPresent])
		}
		if !test.expectErr {
			if err != nil {
				t.Errorf("Test %v (%s) Failed: Received unexpected error - %v", i, test.description, *err)
			} else {
				if test.expectedReqURI != requestURI {
					t.Errorf("Test %v (%s) Failed: Expected request URI is '%v'; Received '%v'", i, test.description, test.expectedReqURI, requestURI)
				}
			}
		} else if err == nil {
			t.Errorf("Test %v (%s) Failed: Expected an error", i, test.description)
		}
	}
}

func Test_verifyCabinetRedfishEndpoints(t *testing.T) {
	type HTTPResponse struct {
		respCode        int
		respBody        string
		expectedReqBody []byte
	}

	tests := []struct {
		description           string
		responses             map[string]HTTPResponse
		netEndpoints          []*NetEndpoint
		redfishEndpointsCache map[string]HSMNotification
		expectErr             bool
	}{{
		description: "Nothing to change",
		responses:   map[string]HTTPResponse{},
		netEndpoints: []*NetEndpoint{{
			name:   "x9000c1b0",
			mac:    "02:23:28:01:00:00",
			hwtype: TYPE_CHASSIS,
		}, {
			name:   "x9000c1r1b0",
			mac:    "02:23:28:01:61:00",
			hwtype: TYPE_SWITCH_CARD,
		}, {
			name:   "x9000c3s0b0",
			mac:    "02:23:28:03:30:00",
			hwtype: TYPE_NODE_CARD,
		}},
		redfishEndpointsCache: map[string]HSMNotification{
			"x9000c1b0": {
				ID:                 "x9000c1b0",
				FQDN:               "x9000c1b0",
				Hostname:           "x9000c1b0",
				User:               "root",
				MACAddr:            "02:23:28:01:00:00",
				RediscoverOnUpdate: true,
			},
			"x9000c1r1b0": {
				ID:                 "x9000c1r1b0",
				FQDN:               "x9000c1r1b0",
				Hostname:           "x9000c1r1b0",
				User:               "root",
				MACAddr:            "02:23:28:01:61:00",
				RediscoverOnUpdate: true,
			},
			"x9000c3s0b0": {
				ID:                 "x9000c3s0b0",
				FQDN:               "x9000c3s0b0",
				Hostname:           "x9000c3s0b0",
				User:               "root",
				MACAddr:            "02:23:28:03:30:00",
				RediscoverOnUpdate: true,
			}},
		expectErr: false,
	}, {
		description: "Chassis BMC with old FQDN",
		responses: map[string]HTTPResponse{
			"/Inventory/RedfishEndpoints/x9000c1b0": {
				respCode:        200,
				respBody:        `{"ID":"x1000c6b0","Type":"ChassisBMC","Hostname":"x1000c6b0","Domain":"","FQDN":"x1000c6b0","Enabled":true,"User":"","Password":"","RediscoverOnUpdate":true,"DiscoveryInfo":{"LastDiscoveryAttempt":"2021-06-30T23:33:46.847154Z","LastDiscoveryStatus":"HTTPsGetFailed"}}`,
				expectedReqBody: json.RawMessage(`{"ID":"x9000c1b0","FQDN":"x9000c1b0","Hostname":"x9000c1b0"}`),
			},
		},
		netEndpoints: []*NetEndpoint{{
			name:   "x9000c1b0",
			mac:    "02:23:28:01:00:00",
			hwtype: TYPE_CHASSIS,
		}, {
			name:   "x9000c1r1b0",
			mac:    "02:23:28:01:61:00",
			hwtype: TYPE_SWITCH_CARD,
		}, {
			name:   "x9000c3s0b0",
			mac:    "02:23:28:03:30:00",
			hwtype: TYPE_NODE_CARD,
		}},
		redfishEndpointsCache: map[string]HSMNotification{
			"x9000c1b0": {
				ID:                 "x9000c1b0",
				FQDN:               "x9000c1",
				Hostname:           "x9000c1",
				User:               "root",
				MACAddr:            "02:23:28:01:00:00",
				RediscoverOnUpdate: true,
			},
			"x9000c1r1b0": {
				ID:                 "x9000c1r1b0",
				FQDN:               "x9000c1r1b0",
				Hostname:           "x9000c1r1b0",
				User:               "root",
				MACAddr:            "02:23:28:01:61:00",
				RediscoverOnUpdate: true,
			},
			"x9000c3s0b0": {
				ID:                 "x9000c3s0b0",
				FQDN:               "x9000c3s0b0",
				Hostname:           "x9000c3s0b0",
				User:               "root",
				MACAddr:            "02:23:28:03:30:00",
				RediscoverOnUpdate: true,
			}},
		expectErr: false,
	}, {
		description: "Mix of old and new Chassis BMC FQDN",
		responses: map[string]HTTPResponse{
			"/Inventory/RedfishEndpoints/x9000c1b0": {
				respCode:        200,
				respBody:        `{"ID":"x1000c6b0","Type":"ChassisBMC","Hostname":"x1000c6b0","Domain":"","FQDN":"x1000c6b0","Enabled":true,"User":"","Password":"","RediscoverOnUpdate":true,"DiscoveryInfo":{"LastDiscoveryAttempt":"2021-06-30T23:33:46.847154Z","LastDiscoveryStatus":"HTTPsGetFailed"}}`,
				expectedReqBody: json.RawMessage(`{"ID":"x9000c1b0","FQDN":"x9000c1b0","Hostname":"x9000c1b0"}`),
			},
		},
		netEndpoints: []*NetEndpoint{{
			name:   "x9000c1b0",
			mac:    "02:23:28:01:00:00",
			hwtype: TYPE_CHASSIS,
		}, {
			name:   "x9000c3b0",
			mac:    "02:23:28:03:00:00",
			hwtype: TYPE_CHASSIS,
		}, {
			name:   "x9000c1r1b0",
			mac:    "02:23:28:01:61:00",
			hwtype: TYPE_SWITCH_CARD,
		}, {
			name:   "x9000c3s0b0",
			mac:    "02:23:28:03:30:00",
			hwtype: TYPE_NODE_CARD,
		}},
		redfishEndpointsCache: map[string]HSMNotification{
			"x9000c1b0": {
				ID:                 "x9000c1b0",
				FQDN:               "x9000c1",
				Hostname:           "x9000c1",
				User:               "root",
				MACAddr:            "02:23:28:01:00:00",
				RediscoverOnUpdate: true,
			},
			"x9000c3b0": {
				ID:                 "x9000c3b0",
				FQDN:               "x9000c3b0",
				Hostname:           "x9000c3b0",
				User:               "root",
				MACAddr:            "02:23:28:03:00:00",
				RediscoverOnUpdate: true,
			},
			"x9000c1r1b0": {
				ID:                 "x9000c1r1b0",
				FQDN:               "x9000c1r1b0",
				Hostname:           "x9000c1r1b0",
				User:               "root",
				MACAddr:            "02:23:28:01:61:00",
				RediscoverOnUpdate: true,
			},
			"x9000c3s0b0": {
				ID:                 "x9000c3s0b0",
				FQDN:               "x9000c3s0b0",
				Hostname:           "x9000c3s0b0",
				User:               "root",
				MACAddr:            "02:23:28:03:30:00",
				RediscoverOnUpdate: true,
			}},
		expectErr: false,
	}, {
		description: "HSM is unavialable",
		responses: map[string]HTTPResponse{
			"/Inventory/RedfishEndpoints/x9000c1b0": {
				respCode:        500,
				respBody:        ``,
				expectedReqBody: json.RawMessage(`{"ID":"x9000c1b0","FQDN":"x9000c1b0","Hostname":"x9000c1b0"}`),
			},
		},
		netEndpoints: []*NetEndpoint{{
			name:   "x9000c1b0",
			mac:    "02:23:28:01:00:00",
			hwtype: TYPE_CHASSIS,
		}, {
			name:   "x9000c3b0",
			mac:    "02:23:28:03:00:00",
			hwtype: TYPE_CHASSIS,
		}, {
			name:   "x9000c1r1b0",
			mac:    "02:23:28:01:61:00",
			hwtype: TYPE_SWITCH_CARD,
		}, {
			name:   "x9000c3s0b0",
			mac:    "02:23:28:03:30:00",
			hwtype: TYPE_NODE_CARD,
		}},
		redfishEndpointsCache: map[string]HSMNotification{
			"x9000c1b0": {
				ID:                 "x9000c1b0",
				FQDN:               "x9000c1",
				Hostname:           "x9000c1",
				User:               "root",
				MACAddr:            "02:23:28:01:00:00",
				RediscoverOnUpdate: true,
			},
			"x9000c3b0": {
				ID:                 "x9000c3b0",
				FQDN:               "x9000c3b0",
				Hostname:           "x9000c3b0",
				User:               "root",
				MACAddr:            "02:23:28:03:00:00",
				RediscoverOnUpdate: true,
			},
			"x9000c1r1b0": {
				ID:                 "x9000c1r1b0",
				FQDN:               "x9000c1r1b0",
				Hostname:           "x9000c1r1b0",
				User:               "root",
				MACAddr:            "02:23:28:01:61:00",
				RediscoverOnUpdate: true,
			},
			"x9000c3s0b0": {
				ID:                 "x9000c3s0b0",
				FQDN:               "x9000c3s0b0",
				Hostname:           "x9000c3s0b0",
				User:               "root",
				MACAddr:            "02:23:28:03:30:00",
				RediscoverOnUpdate: true,
			}},
		expectErr: true,
	}, {
		description: "No redfish endpoints present in HSM",
		responses:   map[string]HTTPResponse{},
		netEndpoints: []*NetEndpoint{{
			name:   "x9000c1b0",
			mac:    "02:23:28:01:00:00",
			hwtype: TYPE_CHASSIS,
		}, {
			name:   "x9000c3b0",
			mac:    "02:23:28:03:00:00",
			hwtype: TYPE_CHASSIS,
		}, {
			name:   "x9000c1r1b0",
			mac:    "02:23:28:01:61:00",
			hwtype: TYPE_SWITCH_CARD,
		}, {
			name:   "x9000c3s0b0",
			mac:    "02:23:28:03:30:00",
			hwtype: TYPE_NODE_CARD,
		}},
		redfishEndpointsCache: map[string]HSMNotification{},
		expectErr:             false,
	}}

	serviceName = "MEDS_TEST"
	defUser = "root"
	defPass = "********"
	client, _ = hms_certs.CreateHTTPClientPair("", clientTimeout)
	setupRFHTTPStuff()

	for i, test := range tests {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath := r.URL.Path
			requestBody, _ := ioutil.ReadAll(r.Body)

			httpr, ok := test.responses[requestPath]
			if ok {
				// Check for User-Agent headers
				if !userAgentHeaderPresent(r) {
					t.Errorf("Test %v had no User-Agent header.", i)
				}

				// Check the request is the one we expected
				if bytes.Compare(httpr.expectedReqBody, requestBody) != 0 {
					t.Errorf("Test %v (%s) Failed: Expected request body is '%v'; Received '%v'", i, test.description, string(httpr.expectedReqBody), string(requestBody))
				}

				w.WriteHeader(httpr.respCode)
				w.Write(json.RawMessage(httpr.respBody))
			} else {
				w.WriteHeader(500)
				w.Write([]byte("Couldn't find HTTPResponse to give for URL " + requestPath))
			}
		}))
		defer testServer.Close()
		hsm = testServer.URL

		// Setup global variables
		hsmRedfishEndpointsCacheLock = sync.Mutex{}
		hsmRedfishEndpointsCache = test.redfishEndpointsCache

		err := (verifyCabinetRedfishEndpoints(test.netEndpoints))
		if !test.expectErr {
			if err != nil {
				t.Errorf("Test %v (%s) Failed: Received unexpected error - %v", i, test.description, err)
			}
		} else if err == nil {
			t.Errorf("Test %v (%s) Failed: Expected an error", i, test.description)
		}
	}
}
