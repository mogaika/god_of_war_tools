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

	"../utils"
)

func Unpack(wad string, outdir string) error {
	f, err := os.OpenFile(wad, os.O_RDONLY, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	log.Printf("Using out dir \"%s\"\n", outdir)
	os.Mkdir(outdir, 0777)
	item := make([]byte, 32)
	data := false

	for {
		rpos, _ := f.Seek(0, os.SEEK_CUR)
		n, err := f.Read(item)
		if err != nil {
			if err == io.EOF {
				if n != 32 && n != 0 {
					return errors.New("File end is corrupt")
				} else {
					return nil
				}
			} else {
				return err
			}
		}

		tag := binary.LittleEndian.Uint16(item[0:2])
		param := binary.LittleEndian.Uint16(item[2:4])
		size := binary.LittleEndian.Uint32(item[4:8])
		name := utils.BytesToString(item[8:32])

		if !data {
			switch tag {
			case 0x378: // file header start
			case 0x28: // file header group start
			case 0x32: // file header group end
			case 0x29a: // file data start
				data = true
			}
		} else {
			switch tag {
			case 0x28: // file data group start
			case 0x32: // file data group end
			case 0x1e: // file data packet
				if size != 0 {
					fname := path.Join(outdir, name)
					log.Printf("Creating file %s\n", fname)
					of, err := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY, 0777)
					if err != nil {
						log.Printf("Cannot open file \"%s\" for writing: %v\n", fname, err)
					} else {
						defer of.Close()
						_, err := io.CopyN(of, f, int64(size))
						if err != nil {
							log.Printf("Error when writing data to file \"%s\":%v\n", fname, err)
						}
					}
				}
			}

			if size == 0 {
				log.Printf("%.8x:%.4x:%.4x: tag %s\n", rpos, tag, param, name)
			} else {
				log.Printf("%.8x:%.4x:%.4x:%.8x data %s\n", rpos, tag, param, size, name)
			}
		}

		off := (size + 15) & (15 ^ math.MaxUint32)
		f.Seek(int64(off)+rpos+32, os.SEEK_SET)
	}
}

func main() {
	flag.Parse()
	args := flag.Args()

	wad := `E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\GODOFWAR_BACKUP\pack\R_HERO0.WAD`
	if len(args) > 0 {
		wad = args[0]
	}

	out := wad + "_unpacked"
	if len(args) > 1 {
		out = args[1]
	}

	if err := Unpack(wad, out); err != nil {
		log.Fatalln("Error when unpaking wad file: ", err)
	}
}
