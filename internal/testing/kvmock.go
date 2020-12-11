// Copyright 2019 Cray Inc. All Rights Reserved.
// Except as permitted by contract or express written permission of Cray Inc.,
// no part of this work or its content may be modified, used, reproduced or
// disclosed in any form. Modifications made without express permission of
// Cray Inc. may damage the system the software is installed within, may
// disqualify the user from receiving support from Cray Inc. under support or
// maintenance contracts, or require additional support services outside the
// scope of those contracts to repair the software or system.

package testing

import (
	"github.com/mitchellh/mapstructure"
)

type KvMock struct {
	storage map[string]interface{}
}

func NewKvMock() *KvMock {
	var s KvMock
	s.storage = make(map[string]interface{})
	return &s
}

func (kv KvMock) Store(key string, value interface{}) error {
	kv.storage[key] = value
	return nil
}
func (kv KvMock) Lookup(key string, output interface{}) error {
	value := kv.storage[key]
	err := mapstructure.Decode(value, output)
	return err
}
func (kv KvMock) Delete(key string) error {
	delete(kv.storage, key)
	return nil
}
func (kv KvMock) LookupKeys(keyPath string) (keys []string, err error) {
	return
}
