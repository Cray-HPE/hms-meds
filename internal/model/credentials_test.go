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
