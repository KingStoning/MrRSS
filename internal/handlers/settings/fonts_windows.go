package settings

import (
	"fmt"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

// LOGFONTW's face name is the CSS family name, unlike registry display names
// which can contain style names and multiple font faces from a collection.
type fontLog struct {
	Height, Width, Escapement, Orientation, Weight       int32
	Italic, Underline, StrikeOut, CharSet                byte
	OutPrecision, ClipPrecision, Quality, PitchAndFamily byte
	FaceName                                             [32]uint16
}

func installedFontFamilies() ([]string, error) {
	user := syscall.NewLazyDLL("user32.dll")
	gdi := syscall.NewLazyDLL("gdi32.dll")
	dc, _, _ := user.NewProc("GetDC").Call(0)
	if dc == 0 {
		return nil, fmt.Errorf("get font device context")
	}
	defer user.NewProc("ReleaseDC").Call(0, dc)
	families := map[string]bool{}
	callback := syscall.NewCallback(func(font, metric, kind, param uintptr) uintptr {
		family := syscall.UTF16ToString((*fontLog)(unsafe.Pointer(font)).FaceName[:])
		if family != "" && !strings.HasPrefix(family, "@") {
			families[family] = true
		}
		return 1
	})
	filter := fontLog{CharSet: 1}
	gdi.NewProc("EnumFontFamiliesExW").Call(dc, uintptr(unsafe.Pointer(&filter)), callback, 0, 0)
	result := make([]string, 0, len(families))
	for family := range families {
		result = append(result, family)
	}
	sort.Strings(result)
	return result, nil
}
