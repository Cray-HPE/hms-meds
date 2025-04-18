// This file contains the code to enable pprof profiling. It is only
// included in the build when the 'pprof' build tag is set in the Dockerfile.
//
//go:build pprof

/*
 * (C) Copyright [2025] Hewlett Packard Enterprise Development LP
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
	"log"
	"net/http"
	_ "net/http/pprof"
)

// Because MEDS does not start an HTTP server we need to create one when
// pprof is enabled.
func PProfInit() {
	log.Printf("Starting pprof HTTP server")

	go func() {
		err := http.ListenAndServe(":6060", nil)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Failed to start pprof HTTP server: %v", err)
		}
		log.Printf("pprof HTTP server stopped")
	}()

	log.Printf("Started pprof HTTP server")
}