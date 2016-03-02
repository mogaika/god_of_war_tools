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
		unk4:     binary.LittleEndian.Uint32(buf[4:8]),
		palsize:  binary.LittleEndian.Uint32(buf[8:12]),
		unkC:     binary.LittleEndian.Uint32(buf[12:16]),
		unk10:    binary.LittleEndian.Uint32(buf[16:20]),
		palcount: binary.LittleEndian.Uint32(buf[20:24]),
	}

	pal.data = make([][]color.RGBA, pal.palcount)

	for palnum := uint32(0); palnum < pal.palcount; palnum++ {
		palbuf := make([]byte, 0x100*4)
		fpal.Read(palbuf)

		pallet := make([]color.RGBA, 0x100)
		for i := 0; i < 0x100; i++ {
			si := i * 4

			clr := color.RGBA{
				R: palbuf[si],
				G: palbuf[si+1],
				B: palbuf[si+2],
				A: 0xff,
			}

			// apply pallet remapping
			blockid := i / 8
			blockpos := i % 8

			remap := []int{0, 2, 1, 3}

			newpos := blockpos + (remap[blockid%4]+(blockid/4)*4)*8

			pallet[newpos] = clr
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

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	log.Printf("Image sizes: %vx%v", width, height)
	data := make([]byte, width*height*4)
	_, err = fgfx.Read(data)
	if err != nil {
		return img, err
	}

	switch encoding {
	case 0:
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
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

	fout, err := os.OpenFile(out+".png", os.O_CREATE|os.O_WRONLY, 0777)
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
