package txr

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path"

	"github.com/mogaika/god_of_war_tools/utils"

	file_gfx "github.com/mogaika/god_of_war_tools/files/gfx"
)

type Texture struct {
	GfxName    string
	PalName    string
	SubTxrName string
}

const FILE_SIZE = 0x58

func NewFromData(fin io.ReaderAt) (*Texture, error) {
	buf := make([]byte, FILE_SIZE)
	if _, err := fin.ReadAt(buf, 0); err != nil {
		return nil, err
	}

	tex_indify := binary.LittleEndian.Uint32(buf[:4])

	if tex_indify != 7 {
		return nil, errors.New("Not txr file. Magic is not valid.")
	}

	tex := &Texture{
		GfxName:    utils.BytesToString(buf[4:28]),
		PalName:    utils.BytesToString(buf[28:52]),
		SubTxrName: utils.BytesToString(buf[52:76]),
	}
	return tex, nil
}

func (txr *Texture) Image(gfx *file_gfx.GFX, pal *file_gfx.GFX, igfx int, ipal int) (image.Image, error) {
	width := int(gfx.Width)
	height := int(gfx.Height)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pallete := pal.GetPallet(ipal)
	data := gfx.Data[igfx]

	log.Printf("Pallette: %s", pal.String())

	switch gfx.Encoding {
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

				img.Set(x, y, pallete[palpos])
			}
		}
	case 2:
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.Set(x, y, pallete[data[x+y*width]])
			}
		}
	}

	return img, nil
}

func (txr *Texture) Extract(gfx *file_gfx.GFX, pal *file_gfx.GFX, out string) ([]string, error) {
	names := make([]string, 0)
	for iGfx := range gfx.Data {
		for iPal := range pal.Data {
			img, err := txr.Image(gfx, pal, iGfx, iPal)
			if err != nil {
				return nil, err
			}

			var resultFileName string
			if iGfx == 0 && iPal == 0 {
				resultFileName = out + ".png"
			} else {
				resultFileName = fmt.Sprintf("%s.%d.%d.png", out, iGfx, iPal)
			}

			err = os.MkdirAll(path.Dir(resultFileName), 0777)
			if err != nil {
				return nil, err
			}

			fof, err := os.Create(resultFileName)
			if err != nil {
				return nil, err
			}
			defer fof.Close()

			if err = png.Encode(fof, img); err != nil {
				return nil, err
			}

			names = append(names, resultFileName)
		}
	}

	return names, nil
}
