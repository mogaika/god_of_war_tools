package utils

import (
	"bytes"
	"os"
	"strings"
)

const SectorSize = 0x800

const (
	GAME_VERSION_UNKNOWN = iota
	GAME_VERSION_GOW_1
	GAME_VERSION_GOW_2
)

func BytesToString(bs []byte) string {
	n := bytes.IndexByte(bs, 0)
	if n < 0 {
		n = len(bs)
	}
	return string(bs[0:n])
}

// path joining/splitting buggy on windows
func PathPrepare(p string) string {
	return strings.Replace(p, string(os.PathSeparator), "/", -1)
}
