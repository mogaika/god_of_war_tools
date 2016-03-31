package gfx

import (
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"log"
	"math"
)

func Decode(fgfx io.Reader, pal color.Palette) (image.Image, error) {
	buf := make([]byte, 24)
	readn, err := fgfx.Read(buf)
	if err != nil && err != io.EOF || (readn != 0 && readn != 24) {
		return nil, err
	}

	width := int(binary.LittleEndian.Uint32(buf[4:8]))
	height := int(binary.LittleEndian.Uint32(buf[8:12]))
	encoding := int(binary.LittleEndian.Uint32(buf[12:16]))
	bpi := int(binary.LittleEndian.Uint32(buf[16:20]))

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	log.Printf("Width: %v Height: %v Bpi: %v Encoding: %v\n", width, height, bpi, encoding)

	data := make([]byte, (width*height*bpi)/8)
	_, err = fgfx.Read(data)
	if err != nil {
		return img, err
	}

	switch bpi {
	case 4:
		newdata := make([]byte, width*height)
		for i, v := range data {
			newdata[i*2] = v & 0xf
			newdata[i*2+1] = (v >> 4) & 0xf
		}
		data = newdata
		encoding = 2
	case 8:
	default:
		return img, errors.New("Unknown gfx bpi")
	}
	switch encoding {
	case 0:
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				// apply swizzle
				block_location := (y&(math.MaxInt32^0xf))*width + (x&(math.MaxInt32^0xf))*2
				swap_selector := (((y + 2) >> 2) & 0x1) * 4
				posY := (((y & (math.MaxInt32 ^ 3)) >> 1) + (y & 1)) & 0x7
				column_location := posY*width*2 + ((x+swap_selector)&0x7)*4

				byte_num := ((y >> 1) & 1) + ((x >> 2) & 2) // 0,1,2,3

				datapos := block_location + column_location + byte_num
				palpos := data[datapos]

				img.Set(x, y, pal[palpos])
			}
		}
	case 2:
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.Set(x, y, pal[data[x+y*width]])
			}
		}
	default:
		return img, errors.New("Unknown texture encoding")
	}

	return img, nil
}
