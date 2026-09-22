package settings

import (
	"sort"
	"testing"
)

func TestInstalledFontFamiliesWindows(t *testing.T) {
	fonts, err := installedFontFamilies()
	if err != nil {
		t.Fatal(err)
	}
	if len(fonts) == 0 {
		t.Fatal("Windows returned no font families")
	}
	if !sort.StringsAreSorted(fonts) {
		t.Fatal("font families are not sorted")
	}
	for i, f := range fonts {
		if f == "" || (i > 0 && fonts[i-1] == f) {
			t.Fatal("empty or duplicate family")
		}
	}
	t.Logf("Enumerated %d installed Windows font families", len(fonts))
}
