package utils

import "bytes"

const SectorSize = 0x800

const (
	GAME_VERSION_UNKNOWN = iota
	GAME_VERSION_GOW_1_1DVD
	GAME_VERSION_GOW_2_1DVD
)

func BytesToString(bs []byte) string {
	n := bytes.IndexByte(bs, 0)
	if n < 0 {
		n = len(bs)
	}
	return string(bs[0:n])
}
