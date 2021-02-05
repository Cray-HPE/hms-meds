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
	"fmt"
	"path"

	sstorage "stash.us.cray.com/HMS/hms-securestorage"
)

// CredentialsKeyPrefix is the base of the Vault key for credentials
//   This is the default and the actual one comes from values.yaml
const CredentialsKeyPrefix = "meds-cred"

// CredentialsGlobalKey is the Vault key used to access MEDS global
//   credentials
const CredentialsGlobalKey = "global/ipmi"

// Vault Key used for BMC SSH key info
const CredentialsSSHKey = "bmc-ssh-creds"

// A MedsCredStore holds the connection to a Vault and the base path
//   used to formulate keys
type MedsCredStore struct {
	CCPath string
	SS     sstorage.SecureStorage
}

// A MedsCredentials represents a username/password pair
type MedsCredentials struct {
	Username string `json:"Username"`
	Password string `json:"Password"`
}

// BMC SSH creds
type MedsSSHCredentials struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	AuthorizedKey string `json:"authorizedkey"`
}

////////////////////// Global/MEDS creds /////////////////////////////////

// Due to the sensitive nature of the data in MedsCredentials, make a custom String function
// to prevent passwords from being printed directly (accidentally) to output.
func (medsCred MedsCredentials) String() string {
	return fmt.Sprintf("Username: %s, Password: <REDACTED>", medsCred.Username)
}

// Create a new MedsCredStore struct that uses a SecureStorage backing store.
func NewMedsCredStore(keyPath string, ss sstorage.SecureStorage) (mcs *MedsCredStore) {
	mcs = &MedsCredStore{
		CCPath: keyPath,
		SS:     ss,
	}

	return
}

// Fetch the global credentials for Mountain blade BMCs from Vault.
func (mcs *MedsCredStore) FindGlobalCredentials() (medsCred MedsCredentials, err error) {
	err = mcs.SS.Lookup(path.Join(mcs.CCPath, CredentialsGlobalKey), &medsCred)
	return
}

// Store the global credentials for Mountain blade BMCs into Vault.
func (mcs *MedsCredStore) StoreGlobalCredentials(medsCred MedsCredentials) (err error) {
	err = mcs.SS.Store(path.Join(mcs.CCPath, CredentialsGlobalKey), medsCred)
	return
}

/////////////////////////////// BMC SSH CREDS ////////////////////////////

// Fetch BMC SSH creds.  These can be all the same/global, or can be indexed by
// XName.  If the 'xname' parameter is empty, we'll assume global.

func (mcs *MedsCredStore) FindBMCSSHCredentials(xname string) (sshCreds MedsSSHCredentials, err error) {
	if xname == "" {
		err = mcs.SS.Lookup(path.Join(mcs.CCPath, CredentialsKeyPrefix), &sshCreds)
	} else {
		err = mcs.SS.Lookup(path.Join(mcs.CCPath, CredentialsKeyPrefix, CredentialsSSHKey, xname), &sshCreds)
	}
	return
}
