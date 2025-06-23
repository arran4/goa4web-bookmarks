package a4webbm

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

const complexBookmarkText = `Category: Example 1
http://www.google.com.au Google
Column
Category: Example 2
http://www.google.com.au Google
http://www.google.com.au Google

Category
http://www.google.com.au Google

Page: Test

Category
http://www.google.com.au Google

Category: Example
http://www.google.com.au Google
http://www.google.com.au Google

Tab

Category
http://www.google.com.au Google

Tab: asdf
Category
http://www.google.com.au Google
`

func TestSerializeBookmarksRoundTrip(t *testing.T) {
	samples := []string{
		defaultBookmarks,
		complexBookmarkText,
		multiBookmarkText,
	}
	for _, in := range samples {
		tabs1 := ParseBookmarks(in)
		out := tabs1.String()
		tabs2 := ParseBookmarks(out)
		if diff := cmp.Diff(tabs1, tabs2); diff != "" {
			t.Fatalf("round trip diff:\n%s", diff)
		}
	}
}
