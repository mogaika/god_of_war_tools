package obj

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/mogaika/god_of_war_tools/files/wad"
	"github.com/mogaika/god_of_war_tools/utils"
)

type Joint struct {
	Name     string
	subStart uint16
	subEnd   uint16
	parent   uint16
	Id       uint16

	SubJoints []*Joint
}

type Object struct {
	Joints []*Joint
}

const OBJECT_MAGIC = 0x00040001
const HEADER_SIZE = 0x2C

func init() {
	wad.PregisterExporter(OBJECT_MAGIC, &Object{})
}

func (j *Joint) String(prefix string) string {
	str := fmt.Sprintf("%s%.2x %s", prefix, j.Id, j.Name)
	if len(j.SubJoints) != 0 {
		str += " {\n"
		pofix := prefix + "  "
		for _, i := range j.SubJoints {
			str += i.String(pofix) + "\n"
		}
		return str + prefix + "}"
	} else {
		return str
	}
}

func NewFromData(rdr io.ReaderAt) (*Object, error) {
	var file [HEADER_SIZE]byte
	_, err := rdr.ReadAt(file[:], 0)
	if err != nil {
		return nil, err
	}

	obj := new(Object)

	log.Printf("     OBJ: %f %f %f     %f %f %f",
		math.Float32frombits(binary.LittleEndian.Uint32(file[0x4:0x8])),
		math.Float32frombits(binary.LittleEndian.Uint32(file[0x8:0xc])),
		math.Float32frombits(binary.LittleEndian.Uint32(file[0xc:0x10])),
		math.Float32frombits(binary.LittleEndian.Uint32(file[0x10:0x14])),
		math.Float32frombits(binary.LittleEndian.Uint32(file[0x14:0x18])),
		math.Float32frombits(binary.LittleEndian.Uint32(file[0x18:0x1c])))

	obj.Joints = make([]*Joint, binary.LittleEndian.Uint32(file[0x1c:0x20]))

	for i := range obj.Joints {
		var jointBuf [0x10]byte
		var nameBuf [0x18]byte

		_, err = rdr.ReadAt(jointBuf[:], int64(HEADER_SIZE+i*0x10))
		if err != nil {
			return nil, err
		}
		_, err = rdr.ReadAt(nameBuf[:], int64(HEADER_SIZE+len(obj.Joints)*0x10+i*0x18))
		if err != nil {
			return nil, err
		}

		obj.Joints[i] = &Joint{
			Name:      utils.BytesToString(nameBuf[:]),
			subStart:  binary.LittleEndian.Uint16(jointBuf[0x4:0x6]),
			subEnd:    binary.LittleEndian.Uint16(jointBuf[0x6:0x8]),
			parent:    binary.LittleEndian.Uint16(jointBuf[0x8:0xa]),
			Id:        uint16(i),
			SubJoints: make([]*Joint, 0)}
	}

	for _, j := range obj.Joints {
		if j.parent < 0x4000 {
			obj.Joints[j.parent].SubJoints = append(obj.Joints[j.parent].SubJoints, j)
		}
	}

	for _, j := range obj.Joints {
		if j.parent > 0x4000 {
			log.Printf("Joints tree:\n%s\n", j.String(""))
		}
	}

	return obj, nil
}

func (*Object) ExtractFromNode(nd *wad.WadNode, outfname string) error {
	log.Printf("Obj '%s' extraction", nd.Path)
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
