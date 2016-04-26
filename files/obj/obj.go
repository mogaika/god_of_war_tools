package obj

import (
	"io"

	"github.com/mogaika/god_of_war_tools/files/wad"
)

type Joint struct {
	Name string
}

type Object struct {
}

const OBJECT_MAGIC = 0x00040001
const HEADER_SIZE = 0x2B

func init() {
	wad.PregisterExporter(OBJECT_MAGIC, &Object{})
}

func NewFromData(rdr io.ReaderAt) (*Object, error) {
	/*
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
	*/
	return nil, nil
}

func (*Object) ExtractFromNode(nd *wad.WadNode, outfname string) error {
	reader, err := nd.DataReader()
	if err != nil {
		return err
	}

	obj, err := NewFromData(reader)
	if err != nil {
		return err
	}

	nd.Cache = obj
	return nil
}
