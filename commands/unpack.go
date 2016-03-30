package commands

import (
	"errors"
	"flag"

	"github.com/mogaika/god_of_war_tools/files/pack"
)

type Unpack struct {
	GameFolder string
	OutFolder  string
	Version    int
}

func (u *Unpack) DefineFlags(f *flag.FlagSet) {
	f.StringVar(&u.GameFolder, "in", "", "*Game folder. (Contains GODOFWAR.TOK file)")
	f.StringVar(&u.OutFolder, "out", "./unpacked", " Directory to store result")
	f.IntVar(&u.Version, "v", pack.TOK_VERSION_UNKNOWN, " Version of game: 0-Auto; 1-GOW1; 2-GOW2")
}

func (u *Unpack) Run() error {
	if u.GameFolder == "" {
		return errors.New("game folder argument required")
	}

	return pack.Unpack(u.GameFolder, u.OutFolder, u.Version)
}
