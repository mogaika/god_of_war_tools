package commands

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/mogaika/god_of_war_tools/files/wad"
	"github.com/mogaika/god_of_war_tools/utils"
)

type Extract struct {
	WadFile   string
	OutFolder string
	Version   int
	Print     bool
	All       bool
	Conv      bool
}

func (u *Extract) DefineFlags(f *flag.FlagSet) {
	f.StringVar(&u.WadFile, "wad", "", "*Wad file")
	f.StringVar(&u.OutFolder, "out", "", " Directory to store result (if empty, not produce files)")
	f.BoolVar(&u.Print, "print", false, " Print user-friendly tree representation of wad file")
	f.BoolVar(&u.Conv, "conv", true, " Convert konwn files")
	f.IntVar(&u.Version, "v", utils.GAME_VERSION_UNKNOWN, " Version of game: 0-Auto; 1-GOW1; 2-GOW2")
}

func (u *Extract) Run() error {
	if u.WadFile == "" {
		return errors.New("Wad file argument required")
	}

	wadfile, err := os.Open(u.WadFile)
	if err != nil {
		return err
	}
	defer wadfile.Close()

	wd, err := wad.NewWad(wadfile, u.Version)
	if err != nil {
		return err
	}

	if u.Print {
		for _, nd := range wd.Nodes {
			fmt.Println(nd)
		}
	}

	if u.OutFolder != "" {
		if err := wd.Extract(u.OutFolder, u.Conv); err != nil {
			return err
		}
	}

	return nil
}
