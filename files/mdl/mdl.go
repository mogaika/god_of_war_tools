package mdl

import (
	"encoding/binary"
	"io"
	"log"
	"math"

	"github.com/mogaika/god_of_war_tools/files/wad"
)

type Model struct {
	TextureCount uint32
}

const MODEL_MAGIC = 0x2000f
const FILE_SIZE = 0x48

func init() {
	wad.PregisterExporter(MODEL_MAGIC, &Model{})
}

func NewFromData(rdr io.ReaderAt) (*Model, error) {
	var file [FILE_SIZE]byte
	_, err := rdr.ReadAt(file[:], 0)
	if err != nil {
		return nil, err
	}

	mdl := new(Model)

	mdl.TextureCount = binary.LittleEndian.Uint32(file[0x14:0x18])

	log.Printf("     MDL: %f  %f  %f",
		math.Float32frombits(binary.LittleEndian.Uint32(file[0x8:0xc])),
		math.Float32frombits(binary.LittleEndian.Uint32(file[0xc:0x10])),
		math.Float32frombits(binary.LittleEndian.Uint32(file[0x10:0x14])))

	return mdl, nil
}

func (*Model) ExtractFromNode(nd *wad.WadNode, outfname string) error {
	reader, err := nd.DataReader()
	if err != nil {
		return err
	}

	mdl, err := NewFromData(reader)
	if err != nil {
		return err
	}

	nd.Cache = mdl
	return nil
}
