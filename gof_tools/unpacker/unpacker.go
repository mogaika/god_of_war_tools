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

const (
	TOK_VERSION_UNKNOWN = iota
	TOK_VERSION_GOW_1_1DVD
	TOK_VERSION_GOW_2_1DVD
)

// Return 1 for god of war1 1-dvd
// Return 2 for god of war2 1-dvd
func DetectTokVersion(tok_file string) (int, error) {
	file, err := os.OpenFile(tok_file, os.O_RDONLY, 0777)
	if err != nil {
		return TOK_VERSION_UNKNOWN, err
	}

	defer file.Close()

	buffer := make([]byte, 4)
	_, err = file.Read(buffer)
	if err != nil {
		return TOK_VERSION_UNKNOWN, err
	}

	ver := TOK_VERSION_GOW_1_1DVD
	strend := false
	for _, i := range buffer {
		if i == 0 {
			strend = true
		} else if i < 20 || i > 127 {
			ver = TOK_VERSION_GOW_2_1DVD
		} else if strend {
			ver = TOK_VERSION_GOW_2_1DVD
			break
		}
	}

	return ver, nil
}

// GOF 1
func ParseTok1(tok_file string) ([]*File, error) {
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
			Pack:     0,
			Size:     binary.LittleEndian.Uint32(buffer[16:20]),
			StartSec: binary.LittleEndian.Uint32(buffer[20:24])}

		files = append(files, file)
	}
}

// GOF 2
func ParseTok2(tok_file string) ([]*File, error) {
	const SectorsInFile = (0x3FFFF800 / SectorSize)
	file, err := os.OpenFile(tok_file, os.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, 4)

	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}

	fcount := binary.LittleEndian.Uint32(buffer)
	maxindex := uint32(0)

	buffer = make([]byte, 36)
	files := make([]*File, 0)

	for i := uint32(0); i < fcount; i++ {
		_, err := file.Read(buffer)
		if err != nil {
			return nil, err
		}

		file := &File{
			Name:     utils.BytesToString(buffer[0:24]),
			Size:     binary.LittleEndian.Uint32(buffer[24:28]),
			StartSec: binary.LittleEndian.Uint32(buffer[32:36])}

		files = append(files, file)
		if file.StartSec > maxindex {
			maxindex = file.StartSec
		}
	}

	buffer = make([]byte, 4)
	posmap := make([]uint32, maxindex+1)
	for i := range posmap {
		_, err := file.Read(buffer)
		if err != nil {
			return nil, err
		}
		sz := binary.LittleEndian.Uint32(buffer)
		posmap[i] = sz
	}

	for _, f := range files {
		pos := posmap[f.StartSec]
		f.StartSec = pos % SectorsInFile
		f.Pack = pos / SectorsInFile
	}

	return files, nil
}

func getPackName(game_folder string, pack uint32) string {
	return path.Join(game_folder, "part"+strconv.Itoa(int(pack+1))+".pak")
}

func Unpack(game_folder string, out_folder string, version int) error {
	var err error
	tok_fname := path.Join(game_folder, "GODOFWAR.TOC")

	os.Mkdir(out_folder, 0777)

	if version == TOK_VERSION_UNKNOWN {
		version, err = DetectTokVersion(tok_fname)
		if err != nil {
			return err
		}
		log.Printf("Detected tok version: %v\n", version)
	}

	var files []*File

	switch version {
	case TOK_VERSION_GOW_1_1DVD:
		files, err = ParseTok1(tok_fname)
		if err != nil {
			return err
		}
	case TOK_VERSION_GOW_2_1DVD:
		files, err = ParseTok2(tok_fname)
		if err != nil {
			return err
		}
	case TOK_VERSION_UNKNOWN:
		return errors.New("Unknown tok version for parsing")
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

			log.Printf("[%.4d/%.4d] Unpaking (pk: %v beg:%.8x sz:%.8x) %s \n", i, len(files), f.Pack+1, f.StartSec*SectorSize, f.Size, f.Name)

			wrtd, err := io.CopyN(fo, disk, int64(f.Size))
			if err == io.EOF && wrtd != int64(f.Size) {
				log.Printf("Parted file: %.8x size: %.8x file: %#v \n", wrtd, f.Size, f)
				curpart++

				disk.Close()
				disk, err = os.OpenFile(getPackName(game_folder, f.Pack), os.O_RDONLY, 0777)
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

func main() {
	flag.Parse()
	args := flag.Args()

	game := `E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\GODOFWAR_BACKUP`
	if len(args) > 0 {
		game = args[0]
	}

	out := path.Join(game, "pack")
	if len(args) > 1 {
		out = args[1]
	}

	if err := Unpack(game, out, TOK_VERSION_UNKNOWN); err != nil {
		log.Fatalln("Error when unpaking: ", err)
	}
}
