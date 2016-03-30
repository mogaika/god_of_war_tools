package pack

import (
	"io"
	"log"
	"math"
	"os"
	"path"
	"strconv"

	"github.com/mogaika/god_of_war_tools/files/tok"
	"github.com/mogaika/god_of_war_tools/utils"
)

func getPackName(game_folder string, pack uint32) string {
	return path.Join(game_folder, "part"+strconv.Itoa(int(pack+1))+".pak")
}

func Unpack(game_folder string, out_folder string, tokfiles tok.TokFile, version int) (err error) {
	os.MkdirAll(out_folder, 0666)

	// Check pack* files for existing
	packpresents := make(map[uint32]bool, 0)
	for _, f := range tokfiles {
		if _, ok := packpresents[f.Pack]; !ok {
			if fl, err := os.Open(getPackName(game_folder, f.Pack)); err == nil {
				packpresents[f.Pack] = true
				fl.Close()
			} else {
				packpresents[f.Pack] = false
			}
		}
	}

	var curpart uint32 = math.MaxUint32
	var disk *os.File

	defer func() {
		if curpart != math.MaxUint32 {
			disk.Close()
		}
	}()

	i := 0
	for name, f := range tokfiles {
		i++
		if packpresents[f.Pack] {
			if f.Pack != curpart {
				newdisk, err := os.Open(getPackName(game_folder, f.Pack))
				if err != nil {
					return err
				}

				disk.Close()
				curpart = f.Pack
				disk = newdisk
			}

			fo, err := os.Create(path.Join(out_folder, name))
			if err != nil {
				return err
			}
			defer fo.Close()

			disk.Seek(int64(f.StartSec*utils.SectorSize), os.SEEK_SET)

			log.Printf("[%.4d/%.4d] Unpaking (pk: %v beg:%.8x sz:%.8x) %s \n", i, len(tokfiles), f.Pack+1, f.StartSec*utils.SectorSize, f.Size, name)

			wrtd, err := io.CopyN(fo, disk, int64(f.Size))
			if err == io.EOF && wrtd != int64(f.Size) {
				log.Printf("Parted file: %.8x size: %.8x file: %#v \n", wrtd, f.Size, f)
				curpart++

				disk.Close()
				disk, err = os.Open(getPackName(game_folder, f.Pack))
				if err != nil {
					return err
				}

				next_wrtd, err := io.CopyN(fo, disk, int64(f.Size)-wrtd)
				if err != nil || (err == io.EOF && (next_wrtd+wrtd) != int64(f.Size)) {
					return err
				}
			} else if err != nil {
				return err
			}

			fo.Close()
		}
	}

	return nil
}
