package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
)

type MDL1_Head struct {
	Magic          uint32
	CommentsOffset uint32
	Trash          []byte
	DataOffset     uint32

	Objects []*MDL1_Object
}

type MDL1_Object struct {
}

func Convert1(mdl string, out string) error {
	f, err := os.OpenFile(mdl, os.O_RDONLY, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	buf_head := make([]byte, 84)
	if _, err := f.Read(buf_head); err != nil {
		return err
	}

	head := &MDL1_Head{
		Magic:          binary.LittleEndian.Uint32(buf_head[0:4]),
		CommentsOffset: binary.LittleEndian.Uint32(buf_head[4:8]),
		DataOffset:     binary.LittleEndian.Uint32(buf_head[80:84]),
	}

	if head.Magic != 0x1000f {
		return fmt.Errorf("Unknown mdl type")
	}

	if _, err := f.Seek(int64(head.DataOffset), os.SEEK_SET); err != nil {
		return err
	}

	for end := false; !end; {
		buf4 := make([]byte, 4)
		n, err := f.Read(buf4)
		if err == io.EOF && n == 0 {
			break
		} else if err != nil {
			return err
		}

		block := binary.LittleEndian.Uint32(buf4)
		blockid := (block >> 24) & 0xff

		size := uint32(math.MaxUint32)

		switch blockid {
		case 0x05:
		case 0x65:
		case 0x6c:
		case 0x6d:
		case 0x6e:
		case 0xd8:
		default:
			if _, err := f.Read(buf4); err != nil {
				return err
			}
			size = binary.LittleEndian.Uint32(buf4)
		}

		if block == 0x10001 {
			// size not -= 4
		} else if blockid == 0x05 {
			size = 20
		} else if blockid == 0x30 {
			size -= 0x20
			size -= 4
		} else if blockid == 0x65 {
			count := (block >> 16) & 0xff
			log.Printf("Count of uv : %v\n", count)
			size = count * 4
		} else if blockid == 0x6d {
			count := (block >> 16) & 0xff
			log.Printf("Count of xyz : %v\n", count)
			size = count*8 + 4
		} else if blockid == 0x6e {
			count := (block >> 16) & 0xff
			log.Printf("Count of colorblend? : %v\n", count)
			size = count*4 + 4
		} else if blockid == 0x6c {
			size = 16
		} else {
			size -= 4
		}

		log.Printf("block 0x%.8x [%.2x]: size 0x%.8x\n", block, blockid, size)

		if _, err := f.Seek(int64(size), os.SEEK_CUR); err != nil {
			return err
		}
	}

	//log.Printf("%#v\n", head)

	return nil
}

func Convert2(mdl string, out string) error {
	f, err := os.OpenFile(mdl, os.O_RDONLY, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func main() {
	flag.Parse()
	args := flag.Args()

	var mdl string

	if len(args) > 0 {
		mdl = args[0]
	} else {
		log.Fatalln("Missed argument")
	}

	out := mdl + ".obj"
	if len(args) > 1 {
		out = args[1]
	}

	if err := Convert1(mdl, out); err != nil {
		log.Fatalln("Error when converting obj file: ", err)
	}
}
