package commands

import (
	"errors"
	"flag"
	"image/color"
	"image/png"
	"log"
	"os"
	"path"

	"github.com/mogaika/god_of_war_tools/files/gfx"
	"github.com/mogaika/god_of_war_tools/files/pal"
	"github.com/mogaika/god_of_war_tools/utils"
)

type Image struct {
	GfxFile string
	PalFile string
	OutFile string
}

func (u *Image) DefineFlags(f *flag.FlagSet) {
	f.StringVar(&u.GfxFile, "gfx", "", "*input file")
	f.StringVar(&u.OutFile, "out", "", " result file (default GFX_file->file.png)")
	f.StringVar(&u.PalFile, "pal", "", " custom pallete file (default GFX_file->PAL_file)")
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

func (u *Image) Run() error {
	if u.GfxFile == "" {
		return errors.New("gfx argument required")
	}

	fdir, fname := path.Split(utils.PathPrepare(u.GfxFile))

	if u.PalFile == "" {
		if fname[0:4] == "GFX_" {
			u.PalFile = path.Join(fdir, "PAL_"+fname[4:len(fname)])
			log.Printf("Generated pallete filename: \"%s\"\n", u.PalFile)
		} else {
			return errors.New("Cannot get pallete file from this GFX file (not start with GFX_)")
		}
	}

	if u.OutFile == "" {
		if fname[0:4] == "GFX_" {
			u.OutFile = path.Join(fdir, fname[4:len(fname)])
		} else {
			u.OutFile = path.Join(fdir, fname)
		}
		u.OutFile += ".png"
		log.Printf("Output filename: \"%s\"\n", u.OutFile)
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

	fgfx, err := os.Open(u.GfxFile)
	if err != nil {
		return err
	}
	defer fgfx.Close()

	img, err := gfx.Decode(fgfx, pals[0])
	if err != nil {
		return err
	}

	fout, err := os.Create(u.OutFile)
	if err != nil {
		return err
	}

	return png.Encode(fout, img)
}
