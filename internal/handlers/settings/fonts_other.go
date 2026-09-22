//go:build !windows

package settings

import "fmt"

func installedFontFamilies() ([]string, error) {
	return nil, fmt.Errorf("use browser font access on this platform")
}
