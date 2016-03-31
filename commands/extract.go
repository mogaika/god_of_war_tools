package commands

import (
	"errors"
	"flag"
	"os"

	"github.com/mogaika/god_of_war_tools/files/wad"
	"github.com/mogaika/god_of_war_tools/utils"
)

type Extract struct {
	WadFile   string
	OutFolder string
	Version   int
}

func (u *Extract) DefineFlags(f *flag.FlagSet) {
	f.StringVar(&u.WadFile, "wad", "", "*Wad file")
	f.StringVar(&u.OutFolder, "out", "./extracted", " Directory to store result")
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

	return wad.Unpack(wadfile, u.OutFolder, u.Version)
}
