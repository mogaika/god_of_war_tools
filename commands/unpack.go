package commands

import (
	"errors"
	"flag"
	"log"
	"os"
	"path"

	"github.com/mogaika/god_of_war_tools/files/pack"
	"github.com/mogaika/god_of_war_tools/files/tok"
	"github.com/mogaika/god_of_war_tools/utils"
)

type Unpack struct {
	GameFolder string
	OutFolder  string
	Version    int
	TokFile    string
	DoList     bool
}

func (u *Unpack) DefineFlags(f *flag.FlagSet) {
	f.StringVar(&u.GameFolder, "in", "", "*Game folder. (Contains GODOFWAR.TOK file)")
	f.BoolVar(&u.DoList, "l", false, " Not unpack, only list files in pack archive")
	f.StringVar(&u.OutFolder, "out", "./unpacked", " Directory to store result")
	f.IntVar(&u.Version, "v", utils.GAME_VERSION_UNKNOWN, " Version of game: 0-Auto; 1-GOW1; 2-GOW2")
	f.StringVar(&u.TokFile, "tok", "", " Custom tok file name (default is \"GODOFWAR.TOK\" in game folder)")
}

func (u *Unpack) Run() error {
	if u.GameFolder == "" {
		return errors.New("game folder argument required")
	}

	if u.TokFile == "" {
		u.TokFile = path.Join(u.GameFolder, "GODOFWAR.TOC")
	}

	tokfile, err := os.Open(u.TokFile)
	if err != nil {
		return err
	}
	defer tokfile.Close()

	tokdata, err := tok.Decode(tokfile, u.Version)
	if err != nil {
		return err
	}

	if u.DoList {
		for name, v := range tokdata {
			log.Printf("pack: %d name: \"%s\" size: %d dups: %d", v.Pack, name, v.Size, v.Count)
		}
		return nil
	}

	return pack.Unpack(u.GameFolder, u.OutFolder, tokdata, u.Version)
}
