package commands

import (
	"errors"
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/mogaika/god_of_war_tools/files/gfx"
	"github.com/mogaika/god_of_war_tools/files/pal"
	"github.com/mogaika/god_of_war_tools/files/txr"
)

type Image struct {
	TxrFile string
	GfxFile string
	PalFile string
	OutFile string
}

func (u *Image) DefineFlags(f *flag.FlagSet) {
	f.StringVar(&u.TxrFile, "txr", "", "*input TXR_ file (in one folder with GFX and PAL files)")
	f.StringVar(&u.OutFile, "out", "", " result file (default TXR_file->file.png)")
	f.StringVar(&u.GfxFile, "gfx", "", " custom gfx file")
	f.StringVar(&u.PalFile, "pal", "", " custom pallete file")
}

func ReadPaletts(palfile string) ([]color.Palette, error) {
	fpal, err := os.Open(palfile)
	if err != nil {
		return nil, err
	}
	defer fpal.Close()

	pals, err := pal.Decode(fpal)
	if err != nil {
		return nil, err
	}

	return pals, err
}

func ReadGfx(gfxfile string, pal color.Palette) (image.Image, error) {
	fgfx, err := os.Open(gfxfile)
	if err != nil {
		return nil, err
	}
	defer fgfx.Close()

	img, err := gfx.Decode(fgfx, pal)
	if err != nil {
		return nil, err
	}

	return img, err
}

func (u *Image) Run() error {
	if u.TxrFile == "" {
		return errors.New("txr argument required")
	}

	if u.OutFile == "" {
		if u.TxrFile[:4] == "TXR_" {
			u.OutFile = u.TxrFile[4:len(u.TxrFile)]
		} else {
			u.OutFile = u.TxrFile
		}
		u.OutFile += ".png"
	}

	ftxr, err := os.Open(u.TxrFile)
	if err != nil {
		return err
	}
	defer ftxr.Close()

	texture, err := txr.Decode(ftxr)
	if err != nil {
		return err
	}

	if u.PalFile == "" {
		u.PalFile = texture.PalName
	}
	if u.GfxFile == "" {
		u.GfxFile = texture.GfxName
	}

	pals, err := ReadPaletts(u.PalFile)
	if err != nil {
		return err
	}

	if len(pals) == 0 {
		return errors.New("palett actually not contain paletts (or bad format parsing)")
	} else if len(pals) > 1 {
		log.Printf("Used only first pallet (total %v pallets)", len(pals))
	}

	img, err := ReadGfx(u.GfxFile, pals[0])
	if err != nil {
		return err
	}

	fout, err := os.Create(u.OutFile)
	if err != nil {
		return err
	}

	return png.Encode(fout, img)
}
