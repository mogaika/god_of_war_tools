package commands

import (
	"errors"
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path"

	"github.com/mogaika/god_of_war_tools/files/gfx"
	"github.com/mogaika/god_of_war_tools/files/pal"
	"github.com/mogaika/god_of_war_tools/files/txr"
)

type Image struct {
	TxrFile        string
	GfxFile        string
	PalFile        string
	OutFile        string
	DoInfo         bool
	NotSubTextures bool
}

func (u *Image) DefineFlags(f *flag.FlagSet) {
	f.StringVar(&u.TxrFile, "txr", "", "*input TXR_ file (in one folder with GFX and PAL files)")
	f.StringVar(&u.OutFile, "out", "", " result file (default TXR_file->file.png)")
	f.StringVar(&u.GfxFile, "gfx", "", " custom gfx file")
	f.StringVar(&u.PalFile, "pal", "", " custom pallete file")
	f.BoolVar(&u.DoInfo, "print", false, " only print info about texture")
	f.BoolVar(&u.NotSubTextures, "nosub", false, " not convert sub-textures (LODS)")
}

func ProcessImage(palfile, gfxfile, outfname string) ([]color.Palette, image.Image, error) {
	fpal, err := os.Open(palfile)
	if err != nil {
		return nil, nil, err
	}
	defer fpal.Close()

	pals, err := pal.Decode(fpal)
	if err != nil {
		return nil, nil, err
	}

	if len(pals) == 0 {
		return nil, nil, errors.New("palett actually not contain paletts (or bad format parsing)")
	} else if len(pals) > 1 {
		log.Printf("Used only first pallet (total %v pallets)", len(pals))
	}

	fgfx, err := os.Open(gfxfile)
	if err != nil {
		return nil, nil, err
	}
	defer fgfx.Close()

	img, err := gfx.Decode(fgfx, pals[0])
	if err != nil {
		return nil, nil, err
	}

	fout, err := os.Create(outfname)
	if err != nil {
		return pals, img, err
	}

	err = png.Encode(fout, img)
	if err != nil {
		return pals, img, err
	}

	return pals, img, err
}

func LoadTxr(fname string) (*txr.Texture, error) {
	ftxr, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer ftxr.Close()

	return txr.Decode(ftxr)
}

func PrintInfo(texture *txr.Texture) {
	log.Printf("Gfx file: %s", texture.GfxName)
	log.Printf("Pallete file: %s", texture.PalName)
	log.Printf("Lod file: %s", texture.SubTxrName)
}

func (u *Image) Run() error {
	if u.TxrFile == "" {
		return errors.New("txr argument required")
	}

	dir, filename := path.Split(u.TxrFile)

	if u.OutFile == "" {
		if filename[:4] == "TXR_" {
			u.OutFile = u.TxrFile[4:len(u.TxrFile)]
		} else {
			u.OutFile = u.TxrFile
		}
		u.OutFile += ".png"
	}

	texture, err := LoadTxr(u.TxrFile)
	if err != nil {
		return err
	}

	if u.DoInfo {
		PrintInfo(texture)
	} else {
		if texture.PalName == "" {
			return errors.New("Texture not have pallete")
		}

		if texture.GfxName == "" {
			return errors.New("Texture not have gfx")
		}

		if u.PalFile == "" {
			u.PalFile = path.Join(dir, texture.PalName)
		}
		if u.GfxFile == "" {
			u.GfxFile = path.Join(dir, texture.GfxName)
		}

		_, _, err := ProcessImage(u.PalFile, u.GfxFile, u.OutFile)
		if err != nil {
			return err
		}
	}
	if !u.NotSubTextures {
		for texture.SubTxrName != "" {
			outfname := texture.SubTxrName

			log.Printf("Finded sub-texture %s", texture.SubTxrName)
			txrfile := path.Join(dir, texture.SubTxrName)
			texture, err = LoadTxr(txrfile)
			if err != nil {
				return err
			}

			if u.DoInfo {
				PrintInfo(texture)
			} else {
				gfxfile := path.Join(path.Dir(u.GfxFile), texture.GfxName)

				if outfname == "TXR_" {
					outfname = outfname[4:len(outfname)]
				}
				outfname += ".png"

				_, _, err := ProcessImage(u.PalFile, gfxfile, outfname)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
