// SPDX-License-Identifier: EUPL-1.2
// Copyright (c) 2025 Oliver Andrich

package rewrite

import "bytes"

// binaryCheckLen is the number of bytes to inspect for null bytes.
const binaryCheckLen = 8000

// isBinary reports whether data looks like a binary file
// by checking for null bytes in the first 8000 bytes.
func isBinary(data []byte) bool {
	return bytes.ContainsRune(data[:min(len(data), binaryCheckLen)], 0)
}
