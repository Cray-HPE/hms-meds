// Copyright 2019 Cray Inc. All Rights Reserved.
// Except as permitted by contract or express written permission of Cray Inc.,
// no part of this work or its content may be modified, used, reproduced or
// disclosed in any form. Modifications made without express permission of
// Cray Inc. may damage the system the software is installed within, may
// disqualify the user from receiving support from Cray Inc. under support or
// maintenance contracts, or require additional support services outside the
// scope of those contracts to repair the software or system.

package model

import (
	"reflect"
	"testing"

	mtest "stash.us.cray.com/HMS/hms-meds/internal/testing"
)

func TestMedsCredStore_FindGlobalCredentials(t *testing.T) {
	ss := mtest.NewKvMock()
	credStorage := NewMedsCredStore(CredentialsKeyPrefix, ss)

	testData := MedsCredentials{
		Username: "foo",
		Password: "bar",
	}

	tests := []struct {
		name         string
		ccs          *MedsCredStore
		pushMedsKey  string
		pushMedsCred MedsCredentials
		wantMedsCred MedsCredentials
	}{{
		name:         "EmptyGet",
		ccs:          credStorage,
		pushMedsKey:  "",
		pushMedsCred: MedsCredentials{},
		wantMedsCred: MedsCredentials{},
	}, {
		name:         "BasicGet",
		ccs:          credStorage,
		pushMedsKey:  CredentialsKeyPrefix + "/" + CredentialsGlobalKey,
		pushMedsCred: testData,
		wantMedsCred: testData,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.pushMedsKey) > 0 {
				ss.Store(tt.pushMedsKey, tt.pushMedsCred)
			}
			gotMedsCred, err := tt.ccs.FindGlobalCredentials()

			if err != nil {
				t.Errorf("MedsCredStore.FindGlobalCredentials() err: %s", err)
			}

			if !reflect.DeepEqual(gotMedsCred, tt.wantMedsCred) {
				t.Errorf("MedsCredStore.FindGlobalCredentials() = %v, want %v", gotMedsCred, tt.wantMedsCred)
			}
		})
	}
}

func TestMedsCredentials_String(t *testing.T) {
	tests := []struct {
		name     string
		medsCred MedsCredentials
		want     string
	}{{
		name:     "RedactedOutput",
		medsCred: MedsCredentials{Username: "admin", Password: "terminal0"},
		want:     "Username: admin, Password: <REDACTED>",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.medsCred.String(); got != tt.want {
				t.Errorf("MedsCredentials.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
