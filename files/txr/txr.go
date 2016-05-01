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
	"github.com/mogaika/god_of_war_tools/files/wad"
)

type Texture struct {
	GfxName       string
	PalName       string
	SubTxrName    string
	unkCoeff      int32
	unkMultiplier float32
	unkFlags1     uint16
	unkFlags2     uint16
}

const FILE_SIZE = 0x58
const FILE_MAGIC = 0x00000007

func init() {
	wad.PregisterExporter(FILE_MAGIC, &Texture{})
}

func NewFromData(fin io.ReaderAt) (*Texture, error) {
	buf := make([]byte, FILE_SIZE)
	if _, err := fin.ReadAt(buf, 0); err != nil {
		return nil, err
	}

	magic := binary.LittleEndian.Uint32(buf[:4])

	if magic != FILE_MAGIC {
		return nil, errors.New("Wrong magic.")
	}

	tex := &Texture{
		GfxName:       utils.BytesToString(buf[4:28]),
		PalName:       utils.BytesToString(buf[28:52]),
		SubTxrName:    utils.BytesToString(buf[52:76]),
		unkCoeff:      int32(binary.LittleEndian.Uint32(buf[76:80])),
		unkMultiplier: math.Float32frombits(binary.LittleEndian.Uint32(buf[80:84])),
		unkFlags1:     binary.LittleEndian.Uint16(buf[84:86]),
		unkFlags2:     binary.LittleEndian.Uint16(buf[86:88]),
	}

	if tex.unkCoeff > 0 {
		return nil, fmt.Errorf("Unkonwn coeff %d", tex.unkCoeff)
	}

	// 0 - any; 8000 - alpha channel
	if tex.unkFlags1 != 0 && tex.unkFlags1 != 0x8000 {
		return nil, fmt.Errorf("Unkonwn unkFlags1 0x%.4x != 0", tex.unkFlags1)
	}

	// 1 - mask; 5d - alpha channel; 51 - font
	if tex.unkFlags2 != 1 && tex.unkFlags2 != 0x5d && tex.unkFlags2 != 0x51 {
		return nil, fmt.Errorf("Unkonwn unkFlags2 0x%.4x (0x1,0x5d,0x51)",
			tex.unkFlags1)
	}

	return tex, nil
}

func (txr *Texture) Image(gfx *file_gfx.GFX, pal *file_gfx.GFX, igfx int, ipal int) (image.Image, error) {
	width := int(gfx.Width)
	height := int(gfx.Height)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pallete, err := pal.GetPallet(ipal)

	if err != nil {
		return nil, err
	}

	data := gfx.Data[igfx]

	encoding := gfx.Encoding

	if gfx.Bpi == 4 {
		encoding = 2
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

func (*Texture) ExtractFromNode(nd *wad.WadNode, outfname string) error {
	reader, err := nd.DataReader()
	if err != nil {
		return err
	}

	txr, err := NewFromData(reader)
	if err != nil {
		return err
	}

	if txr.GfxName != "" && txr.PalName != "" {
		gfxnd := nd.Find(txr.GfxName, true)
		palnd := nd.Find(txr.PalName, true)

		if gfxnd == nil || !gfxnd.Extracted || gfxnd.Cache == nil {
			return fmt.Errorf("GFX '%s' not cached", txr.GfxName)
		}
		if palnd == nil || !palnd.Extracted || palnd.Cache == nil {
			return fmt.Errorf("GFX '%s' not cached", txr.PalName)
		}

		resultfiles, err := txr.Extract(
			gfxnd.Cache.(*file_gfx.GFX),
			palnd.Cache.(*file_gfx.GFX), outfname)

		if err != nil {
			return err
		}
		log.Printf("Texture '%s' extracted: %s", nd.Path, resultfiles)

		nd.ExtractedNames = resultfiles
		nd.Cache = txr
	}
	return nil
}
