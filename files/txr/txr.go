package txr

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/mogaika/god_of_war_tools/utils"
)

type Texture struct {
	GfxName string
	PalName string
}

const FILE_SIZE = 0x58

func Decode(fin io.Reader) (*Texture, error) {
	buf := make([]byte, FILE_SIZE)
	n, err := fin.Read(buf)

	if err != nil {
		if err == io.EOF {
			if n != FILE_SIZE {
				return nil, errors.New("Too short txr file.")
			}
		} else {
			return nil, err
		}
	}

	tex_indify := binary.LittleEndian.Uint32(buf[:4])

	if tex_indify != 7 {
		return nil, errors.New("Not txr file. Magic is not valid.")
	}

	tex := &Texture{
		GfxName: utils.BytesToString(buf[4:28]),
		PalName: utils.BytesToString(buf[28:52]),
	}
	return tex, nil
}
