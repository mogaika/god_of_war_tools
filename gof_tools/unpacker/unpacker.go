package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"io"
	"log"
	"math"
	"os"
	"path"
	"strconv"

	"../utils"
)

const SectorSize = 0x800

type File struct {
	Name     string
	Pack     uint32
	Size     uint32
	StartSec uint32
}

func ParseTok(tok_file string) ([]*File, error) {
	file, err := os.OpenFile(tok_file, os.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, 24)
	files := make([]*File, 0)

	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				if n != 0 && n != 24 {
					return nil, errors.New("Bad file format (not 24 stuct array)")
				} else {
					return files, nil
				}
			} else {
				return nil, err
			}
		}

		file := &File{
			Name:     utils.BytesToString(buffer[0:12]),
			Pack:     binary.LittleEndian.Uint32(buffer[12:16]),
			Size:     binary.LittleEndian.Uint32(buffer[16:20]),
			StartSec: binary.LittleEndian.Uint32(buffer[20:24])}

		files = append(files, file)
	}
}

func getPackName(game_folder string, pack uint32) string {
	return path.Join(game_folder, "part"+strconv.Itoa(int(pack+1))+".pak")
}

func Unpack(game_folder string, out_folder string) error {
	files, err := ParseTok(path.Join(game_folder, "GODOFWAR.TOC"))
	if err != nil {
		return err
	}

	packsizes := make(map[uint32]uint64, 0)
	for _, f := range files {
		fsize := uint64(f.Size) + uint64(f.StartSec*SectorSize)
		if ps, ok := packsizes[f.Pack]; !ok || ps < fsize {
			packsizes[f.Pack] = fsize
		}
	}

	packpresents := make(map[uint32]bool, 0)
	for ps := range packsizes {
		if f, err := os.OpenFile(getPackName(game_folder, ps), os.O_RDONLY, 0777); err == nil {
			packpresents[ps] = true
			f.Close()
		}
	}

	for _, f := range files {
		if !packpresents[f.Pack] {
			f.StartSec += uint32(packsizes[0] / SectorSize)
			f.Pack = 0
		}
	}

	var curpart uint32 = math.MaxUint32
	var disk *os.File

	defer func() {
		if curpart != math.MaxUint32 {
			disk.Close()
		}
	}()

	for i, f := range files {
		if packpresents[f.Pack] {
			if f.Pack != curpart {
				newdisk, err := os.OpenFile(getPackName(game_folder, f.Pack), os.O_RDONLY, 0777)
				if err != nil {
					return err
				}

				disk.Close()
				curpart = f.Pack
				disk = newdisk
			}

			fo, err := os.OpenFile(path.Join(out_folder, f.Name), os.O_CREATE|os.O_WRONLY, 0777)
			if err != nil {
				return err
			}
			defer fo.Close()

			disk.Seek(int64(f.StartSec*SectorSize), os.SEEK_SET)

			log.Printf("[%.4d/%.4d] Unpaking %s (pk: %v beg:%v sz:%v)\n", i, len(files), f.Name, f.Pack+1, f.StartSec*SectorSize, f.Size)
			wrtd, err := io.CopyN(fo, disk, int64(f.Size))
			if err != nil && err != io.EOF {
				return err
			} else if wrtd != int64(f.Size) {
				log.Println(wrtd, f.Size)
				return errors.New("File not copied fully")
			}
			fo.Close()
		}
	}

	//part, err := os.OpenFile(getPackName(game_folder, 1), os.O_RDONLY, 0777)

	return nil
}

func main() {
	flag.Parse()
	args := flag.Args()

	game := "../.."
	if len(args) > 0 {
		game = args[0]
	}

	out := path.Join(game, "pack")
	if len(args) > 1 {
		out = args[1]
	}

	if err := Unpack(game, out); err != nil {
		log.Fatalln("Error when unpaking: ", err)
	}
}
