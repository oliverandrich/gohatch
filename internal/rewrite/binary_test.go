// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{name: "text file", data: []byte("hello world\n"), want: false},
		{name: "empty file", data: []byte{}, want: false},
		{name: "go source", data: []byte("package main\n\nfunc main() {}\n"), want: false},
		{name: "binary with null byte", data: []byte{0x89, 0x50, 0x4E, 0x47, 0x00}, want: true},
		{name: "PNG header", data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00}, want: true},
		{name: "null byte in middle", data: []byte("hello\x00world"), want: true},
		{name: "UTF-8 text", data: []byte("Ünïcödé text with ëmöjïs"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isBinary(tt.data))
		})
	}
}
