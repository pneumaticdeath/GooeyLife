package main

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

// The standard FileExtensionFilter only handles simple
// extensinos (e.g. ".rle") but not compound extensions
// like ".rle.txt" that are sometimes the result of
// browsers saving RLE files
type LongExtensionsFileFilter struct {
	storage.FileFilter
	Extensions []string
}

func (filter *LongExtensionsFileFilter) Matches(uri fyne.URI) bool {
	for _, ext := range filter.Extensions {
		if strings.HasSuffix(uri.Name(), ext) {
			return true
		}
	}
	return false
}
