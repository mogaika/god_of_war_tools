package pal

import (
	"encoding/binary"
	"errors"
	"image/color"
	"io"
)

func Decode(fpal io.Reader) ([]color.Palette, error) {
	buf := make([]byte, 24)
	readn, err := fpal.Read(buf)
	if err != nil && err != io.EOF || (readn != 0 && readn != 24) {
		return nil, err
	}

	palsize := binary.LittleEndian.Uint32(buf[4:8])
	palcount := binary.LittleEndian.Uint32(buf[20:24])

	pal := make([]color.Palette, palcount)

	for palnum := uint32(0); palnum < palcount; palnum++ {
		remap := false
		switch palsize {
		case 0x10:
			remap = true
			palsize = 0x100
		case 0x8:
			palsize = 0x10
		default:
			return nil, errors.New("Unknown pallete size")
		}

		palbuf := make([]byte, palsize*4)
		if _, err := fpal.Read(palbuf); err != nil {
			return nil, err
		}

		pallet := make(color.Palette, palsize)
		for i := range pallet {
			si := i * 4

			clr := color.RGBA{
				R: palbuf[si],
				G: palbuf[si+1],
				B: palbuf[si+2],
				A: byte(float32(palbuf[si+3]) * (255.0 / 128.0)),
			}

			if remap {
				// apply pallet remapping
				blockid := i / 8
				blockpos := i % 8

				remap := []int{0, 2, 1, 3}

				newpos := blockpos + (remap[blockid%4]+(blockid/4)*4)*8

				pallet[newpos] = clr
			} else {
				pallet[i] = clr
			}
		}

		pal[palnum] = pallet
	}

	return pal, nil
}
