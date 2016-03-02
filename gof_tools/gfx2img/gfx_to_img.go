package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path"
	"strings"
)

func printHelp() {
	log.Println(`Usage: ./gfx_to_img gfx_file_path --pal [pal_file_path] --o [out_file]`)
}

type PAL struct {
	unk0     uint32
	unk4     uint32
	palsize  uint32
	unkC     uint32
	unk10    uint32
	palcount uint32
	data     [][]color.RGBA
}

func LoadPal(fname string) (*PAL, error) {
	fpal, err := os.OpenFile(fname, os.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}
	defer fpal.Close()

	buf := make([]byte, 24)
	readn, err := fpal.Read(buf)
	if err != nil && err != io.EOF || (readn != 0 && readn != 24) {
		return nil, err
	}

	pal := &PAL{
		unk0:     binary.LittleEndian.Uint32(buf[0:4]),
		palsize:  binary.LittleEndian.Uint32(buf[4:8]),
		unk4:     binary.LittleEndian.Uint32(buf[8:12]),
		unkC:     binary.LittleEndian.Uint32(buf[12:16]),
		unk10:    binary.LittleEndian.Uint32(buf[16:20]),
		palcount: binary.LittleEndian.Uint32(buf[20:24]),
	}

	pal.data = make([][]color.RGBA, pal.palcount)

	for palnum := uint32(0); palnum < pal.palcount; palnum++ {
		remap := false
		switch pal.palsize {
		case 0x10:
			remap = true
			pal.palsize = 0x100
		case 0x8:
			pal.palsize = 0x10
		default:
			return pal, errors.New("Unknown pallete size")
		}

		palbuf := make([]byte, pal.palsize*4)
		if _, err := fpal.Read(palbuf); err != nil {
			return pal, err
		}

		pallet := make([]color.RGBA, pal.palsize)
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

		pal.data[palnum] = pallet
	}

	return pal, nil
}

func ImageFromGfx(fgfxname string, pal []color.RGBA) (image.Image, error) {
	fgfx, err := os.OpenFile(fgfxname, os.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}
	defer fgfx.Close()

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

				col := pal[palpos]

				img.SetRGBA(x, y, col)
			}
		}
	case 2:
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.SetRGBA(x, y, pal[data[x+y*width]])
			}
		}
	default:
		return img, errors.New("Unknown texture encoding")
	}

	return img, nil
}

func Convert(fgfxname string, fpalname string, out string) error {
	pal, err := LoadPal(fpalname)
	if err != nil {
		return err
	}

	if pal.palcount == 0 {
		return errors.New("Pallete contain only 1 array\n")
	}

	img, err := ImageFromGfx(fgfxname, pal.data[0])
	if err != nil {
		return err
	}

	fout, err := os.OpenFile(out+".png", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer fout.Close()

	if err := png.Encode(fout, img); err != nil {
		return err
	}
	return nil
}

func main() {
	flag_pal := flag.String("pal", "", "pallete file")
	flag_out := flag.String("o", "", "pallete file")
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		printHelp()
	} else if len(args) > 0 {
		gfx := strings.Replace(args[0], "\\", "/", -1)

		pal := ""
		out := ""

		if flag_pal != nil && *flag_pal != "" {
			pal = *flag_pal
		} else {
			fdir, fname := path.Split(gfx)

			if fname[0:4] == "GFX_" {
				pal = path.Join(fdir, "PAL_"+fname[4:len(fname)])
			} else {
				log.Fatalf("Cannot get pallete file from this GFX")
			}
		}

		if flag_out != nil && *flag_out != "" {
			out = *flag_out
		} else {
			out = gfx
		}

		if err := Convert(gfx, pal, out); err != nil {
			log.Fatalf("Error when converting texture: %v\n", err)
		}
	}
}
