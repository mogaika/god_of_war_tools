package obj

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/mogaika/god_of_war_tools/files/wad"
	"github.com/mogaika/god_of_war_tools/utils"
)

type Variable struct {
	Name  string
	Value interface{}
	unk   uint32
}

const VARIABLE_MAGIC = 0x00000017

func init() {
	wad.PregisterExporter(VARIABLE_MAGIC, &Variable{})
}

func NewVariableFromData(rdr io.ReaderAt) (*Variable, error) {
	var file [0x20]byte
	_, err := rdr.ReadAt(file[:], 0)
	if err != nil {
		return nil, err
	}

	val := &Variable{
		Name: utils.BytesToString(file[4:28]),
		unk:  binary.LittleEndian.Uint32(file[28:32])}

	// probably part of name
	if val.unk != 0 {
		return nil, fmt.Errorf("Variable nuk not null: %.8x", val.unk)
	}

	switch val.Name {
	case "HeroBreak":

	case "Breakable":
		var floatBuf [4]byte
		_, err := rdr.ReadAt(floatBuf[:], 0)
		if err != nil {
			return nil, err
		}
		val.Value = binary.LittleEndian.Uint32(floatBuf[0:4])

	case "PushPull":
		fallthrough
	case "IO_CSM":
		fallthrough
	case "IO_TandF":
		var intBuf [8]byte
		_, err := rdr.ReadAt(intBuf[:], 0)
		if err != nil {
			return nil, err
		}
		val.Value = []uint32{binary.LittleEndian.Uint32(intBuf[0:4]),
			binary.LittleEndian.Uint32(intBuf[4:8])}

	default:
		return nil, fmt.Errorf("Unknown value name: '%s'", val.Name)
	}

	log.Printf("Value '%s': %v", val.Name, val.Value)

	return val, nil
}

func (*Variable) ExtractFromNode(nd *wad.WadNode, outfname string) error {
	log.Printf("Variable '%s' extraction", nd.Path)
	reader, err := nd.DataReader()
	if err != nil {
		return err
	}

	val, err := NewVariableFromData(reader)
	if err != nil {
		return err
	}

	nd.Cache = val
	return nil
}
