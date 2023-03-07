// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package framework defines the integration and end-to-end test case for cli core
package framework

import (
	"crypto/rand"
	"math/big"
	"os"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// SliceToSet converts the given slice to set type
func SliceToSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{})
	exists := struct{}{}
	for _, ele := range slice {
		set[ele] = exists
	}
	return set
}

// RandomString generates random string of given length
func RandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[int(n.Int64())]
	}
	return string(b)
}

// RandomNumber generates random string of given length
func RandomNumber(length int) string {
	charset := "1234567890"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[int(n.Int64())]
	}
	return string(b)
}

func GetHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Errorf("error while getting user home directory, error:%s", err.Error())
	}
	return home
}
