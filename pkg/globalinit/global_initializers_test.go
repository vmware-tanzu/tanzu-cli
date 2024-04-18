// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package globalinit

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

const initializerName = "my initializer"

func TestRegisterInitializer(t *testing.T) {
	tests := []struct {
		test               string
		triggerFunc        func() bool
		initializationFunc func(io.Writer) error
	}{
		{
			test:               "registering stores the function",
			triggerFunc:        func() bool { return true },
			initializationFunc: func(io.Writer) error { return nil },
		},
	}
	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			// Reset any previous initializers
			initializers = nil
			RegisterInitializer(initializerName, spec.triggerFunc, spec.initializationFunc)

			assert.Equal(1, len(initializers))
			assert.Equal(initializerName, initializers[0].name)

			addr1 := fmt.Sprintf("%p", initializers[0].triggerFunc)
			addr2 := fmt.Sprintf("%p", spec.triggerFunc)
			assert.Equal(addr1, addr2)

			addr1 = fmt.Sprintf("%p", initializers[0].initializationFunc)
			addr2 = fmt.Sprintf("%p", spec.initializationFunc)
			assert.Equal(addr1, addr2)
		})
	}
}

func TestInitializationRequired(t *testing.T) {
	tests := []struct {
		test         string
		triggerFuncs []func() bool
		expected     bool
	}{
		{
			test:         "single true trigger",
			triggerFuncs: []func() bool{func() bool { return true }},
			expected:     true,
		},
		{
			test:         "single false trigger",
			triggerFuncs: []func() bool{func() bool { return false }},
			expected:     false,
		},
		{
			test:         "multiple true triggers",
			triggerFuncs: []func() bool{func() bool { return true }, func() bool { return true }, func() bool { return true }},
			expected:     true,
		},
		{
			test:         "multiple false triggers",
			triggerFuncs: []func() bool{func() bool { return false }, func() bool { return false }, func() bool { return false }},
			expected:     false,
		},
		{
			test:         "mix of true and false triggers",
			triggerFuncs: []func() bool{func() bool { return false }, func() bool { return true }, func() bool { return false }},
			expected:     true,
		},
	}
	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			// Reset any previous initializers
			initializers = nil

			for _, triggerFunc := range spec.triggerFuncs {
				RegisterInitializer(initializerName, triggerFunc, func(io.Writer) error { return nil })
			}

			assert.Equal(spec.expected, InitializationRequired())
		})
	}
}

func TestPerformInitializations(t *testing.T) {
	tests := []struct {
		test          string
		triggerFuncs  []func() bool
		initFuncs     []func(io.Writer) error
		expectError   bool
		expectedOut   []string
		unexpectedOut []string
	}{
		{
			test: "single true trigger",
			triggerFuncs: []func() bool{
				func() bool { return true },
			},
			initFuncs: []func(io.Writer) error{
				func(w io.Writer) error { fmt.Fprintln(w, "1"); return nil },
			},
			expectError:   false,
			expectedOut:   []string{"1"},
			unexpectedOut: []string{"2", "3"},
		},
		{
			test: "single false trigger",
			triggerFuncs: []func() bool{
				func() bool { return false },
			},
			initFuncs: []func(io.Writer) error{
				func(w io.Writer) error { fmt.Fprintln(w, "1"); return nil },
			},
			expectError:   false,
			unexpectedOut: []string{"1", "2", "3"},
		},
		{
			test: "multiple true triggers",
			triggerFuncs: []func() bool{
				func() bool { return true },
				func() bool { return true },
				func() bool { return true },
			},
			initFuncs: []func(io.Writer) error{
				func(w io.Writer) error { fmt.Fprintln(w, "1"); return nil },
				func(w io.Writer) error { fmt.Fprintln(w, "2"); return nil },
				func(w io.Writer) error { fmt.Fprintln(w, "3"); return nil },
			},
			expectError: false,
			expectedOut: []string{"1", "2", "3"},
		},
		{
			test: "multiple false triggers",
			triggerFuncs: []func() bool{
				func() bool { return false },
				func() bool { return false },
				func() bool { return false },
			},
			initFuncs: []func(io.Writer) error{
				func(w io.Writer) error { fmt.Fprintln(w, "1"); return nil },
				func(w io.Writer) error { fmt.Fprintln(w, "2"); return nil },
				func(w io.Writer) error { fmt.Fprintln(w, "3"); return nil },
			},
			expectError:   false,
			unexpectedOut: []string{"1", "2", "3"},
		},
		{
			test: "mix of true and false triggers",
			triggerFuncs: []func() bool{
				func() bool { return false },
				func() bool { return true },
				func() bool { return false },
			},
			initFuncs: []func(io.Writer) error{
				func(w io.Writer) error { fmt.Fprintln(w, "1"); return nil },
				func(w io.Writer) error { fmt.Fprintln(w, "2"); return nil },
				func(w io.Writer) error { fmt.Fprintln(w, "3"); return nil },
			},
			expectError:   false,
			expectedOut:   []string{"2"},
			unexpectedOut: []string{"1", "3"},
		},
		{
			test: "init throws error",
			triggerFuncs: []func() bool{
				func() bool { return true },
			},
			initFuncs: []func(io.Writer) error{
				func(w io.Writer) error { fmt.Fprintln(w, "1"); return fmt.Errorf("error") },
			},
			expectError:   true,
			expectedOut:   []string{"1"},
			unexpectedOut: []string{"2", "3"},
		},
		{
			test: "init success and error",
			triggerFuncs: []func() bool{
				func() bool { return true },
				func() bool { return true },
				func() bool { return true },
			},
			initFuncs: []func(io.Writer) error{
				func(w io.Writer) error { fmt.Fprintln(w, "1"); return nil },
				func(w io.Writer) error { fmt.Fprintln(w, "2"); return fmt.Errorf("error") },
				func(w io.Writer) error { fmt.Fprintln(w, "3"); return nil },
			},
			expectError: true,
			expectedOut: []string{"1", "2", "3"},
		},
	}
	for _, spec := range tests {
		t.Run(spec.test, func(t *testing.T) {
			assert := assert.New(t)

			// Reset any previous initializers
			initializers = nil

			for i := range spec.initFuncs {
				RegisterInitializer(initializerName, spec.triggerFuncs[i], spec.initFuncs[i])
			}

			var buf bytes.Buffer
			err := PerformInitializations(&buf)

			assert.Equal(spec.expectError, err != nil)

			for i := range spec.expectedOut {
				assert.Contains(buf.String(), spec.expectedOut[i])
			}
			for i := range spec.unexpectedOut {
				assert.NotContains(buf.String(), spec.unexpectedOut[i])
			}
		})
	}
}
