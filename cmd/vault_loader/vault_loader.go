// Copyright 2019 Cray Inc. All Rights Reserved.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"stash.us.cray.com/HMS/hms-meds/internal/model"
	securestorage "stash.us.cray.com/HMS/hms-securestorage"
)

func main() {
	var secureStorage securestorage.SecureStorage
	var credStorage *model.MedsCredStore

	defaultCredentials, ok := os.LookupEnv("VAULT_REDFISH_DEFAULTS")
	if !ok {
		panic("Value not set for VAULT_REDFISH_DEFAULTS")
	}

	// Setup Vault. It's kind of a big deal, so we'll wait forever for this to work.
	fmt.Println("Connecting to Vault...")
	for {
		var err error
		// Start a connection to Vault
		if secureStorage, err = securestorage.NewVaultAdapter("secret"); err != nil {
			fmt.Printf("Unable to connect to Vault (%s)...trying again in 5 seconds.\n", err)
			time.Sleep(5 * time.Second)
		} else {
			fmt.Println("Connected to Vault.")
			credStorage = model.NewMedsCredStore(model.CredentialsKeyPrefix, secureStorage)
			break
		}
	}

	var credentials model.MedsCredentials
	err := json.Unmarshal([]byte(defaultCredentials), &credentials)
	if err != nil {
		panic(err)
	}

	// Uncomment the following lines if you want to debug what is getting put into Vault.
	//prettyCredentials, _ := json.MarshalIndent(credentials, "\t", "   ")
	//fmt.Printf("Loading:\n\t%s\n\n", prettyCredentials)

	err = credStorage.StoreGlobalCredentials(credentials)
	if err != nil {
		panic(err)
	}

	fmt.Println("Done.")
}
