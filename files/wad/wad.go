package wad

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"math"
	"os"
	"path"

	"github.com/mogaika/god_of_war_tools/utils"
)

func DetectVersion(file io.Reader) (int, error) {
	buffer := make([]byte, 4)
	_, err := file.Read(buffer)
	if err != nil {
		return utils.GAME_VERSION_UNKNOWN, err
	}

	first_tag := binary.LittleEndian.Uint32(buffer)
	switch first_tag {
	case 0x378:
		return utils.GAME_VERSION_GOW_1_1DVD, nil
	case 0x15:
		return utils.GAME_VERSION_GOW_2_1DVD, nil
	default:
		return utils.GAME_VERSION_UNKNOWN, errors.New("Cannot detect version")
	}
}

func dataPacket(f io.Reader, size uint32, name, outdir string) {
	if size != 0 && name != "" {
		fname := path.Join(outdir, name)
		log.Printf("Creating file %s\n", fname)
		of, err := os.Create(fname)
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

func Unpack(f io.ReadSeeker, outdir string, version int) (err error) {
	if version == utils.GAME_VERSION_UNKNOWN {
		version, err = DetectVersion(f)
		if err != nil {
			return err
		}
		if version == utils.GAME_VERSION_UNKNOWN {
			return errors.New("Unknown version of WAD")
		}
		f.Seek(0, os.SEEK_SET)
	}

	os.Mkdir(outdir, 0666)
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
		//param := binary.LittleEndian.Uint16(item[2:4])
		size := binary.LittleEndian.Uint32(item[4:8])
		name := utils.BytesToString(item[8:32])

		if version == utils.GAME_VERSION_GOW_2_1DVD {
			if !data {
				switch tag {
				case 0x15: // file header start
				case 0x02: // file header group start
				case 0x03: // file header group end
				case 0x16: // file header pop heap
				case 0x13: // file data start
					data = true
				}
			} else {
				switch tag {
				case 0x02: // file data group start
				case 0x03: // file data group end
				case 0x09: // file data mesh ?
					fallthrough
				case 0x01: // file data packet
					dataPacket(f, size, name, outdir)
				}
			}
		} else if version == utils.GAME_VERSION_GOW_1_1DVD {
			if !data {
				switch tag {
				case 0x378: // file header start
				case 0x28: // file header group start
				case 0x32: // file header group end
				case 0x3e7: // file header pop heap
				case 0x29a: // file data start
					data = true
				}
			} else {
				switch tag {
				case 0x18: // entity count
					size = 0
				case 0x28: // file data group start
				case 0x32: // file data group end
				case 0x1e: // file data packet
					dataPacket(f, size, name, outdir)
				}
			}
		}

		//if size == 0 {
		//	log.Printf("%.8x:%.4x:%.4x: tag %s\n", rpos, tag, param, name)
		//} else {
		//	log.Printf("%.8x:%.4x:%.4x:%.8x data %s\n", rpos, tag, param, size, name)
		//}

		off := (size + 15) & (15 ^ math.MaxUint32)
		f.Seek(int64(off)+rpos+32, os.SEEK_SET)
	}
}
