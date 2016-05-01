package obj

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"

	"github.com/mogaika/god_of_war_tools/files/wad"
	"github.com/mogaika/god_of_war_tools/utils"
)

type Joint struct {
	Name     string
	subStart uint16
	subEnd   uint16
	parent   uint16
	Id       uint16

	Matrix [16]float32
	Arr1   [4]float32
	Arr2   [4]float32
	Arr3   [4]float32
	Arr4   [4]float32
	Arr5   [4]float32
	Arr6   [4]float32

	SubJoints []*Joint
}

type Object struct {
	Joints []*Joint

	matOffset uint32

	jointsCount uint32

	// realtive to matOffset
	jointsCount2 uint32
	// [4] float
	unk1arrOffset uint32 // rotation matrix?
	unk2arrOffset uint32
	unk3arrOffset uint32
	unk4arrOffset uint32
	unk5arrOffset uint32 // scale probably
	unk6arrOffset uint32
}

const OBJECT_MAGIC = 0x00040001
const HEADER_SIZE = 0x2C

func init() {
	wad.PregisterExporter(OBJECT_MAGIC, &Object{})
}

func (j *Joint) String(prefix string) string {
	str := fmt.Sprintf("%s%.2x %s {%.3f %.3f %.3f|%.3f %.3f %.3f|%.3f %.3f %.3f|%.3f %.3f %.3f|%.3f %.3f %.3f|%.3f %.3f %.3f}", prefix, j.Id, j.Name,
		j.Arr1[0], j.Arr1[1], j.Arr1[2],
		j.Arr2[0], j.Arr2[1], j.Arr2[2],
		j.Arr3[0], j.Arr3[1], j.Arr3[2],
		j.Arr4[0], j.Arr4[1], j.Arr4[2],
		j.Arr5[0], j.Arr5[1], j.Arr5[2],
		j.Arr6[0], j.Arr6[1], j.Arr6[2])
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

	log.Printf("     OBJ: %.8x %.8x %.8x   %.8x %.8x %.8x",
		binary.LittleEndian.Uint32(file[0x4:0x8]),
		binary.LittleEndian.Uint32(file[0x8:0xc]),
		binary.LittleEndian.Uint32(file[0xc:0x10]),
		binary.LittleEndian.Uint32(file[0x10:0x14]),
		binary.LittleEndian.Uint32(file[0x14:0x18]),
		binary.LittleEndian.Uint32(file[0x18:0x1c]))

	obj.jointsCount = binary.LittleEndian.Uint32(file[0x1c:0x20])
	obj.matOffset = binary.LittleEndian.Uint32(file[0x28:0x2c])

	obj.Joints = make([]*Joint, obj.jointsCount)

	for i := range obj.Joints {
		var jointBuf [0x10]byte
		var nameBuf [0x18]byte

		_, err = rdr.ReadAt(jointBuf[:], int64(HEADER_SIZE+i*0x10))
		if err != nil {
			return nil, err
		}
		_, err = rdr.ReadAt(nameBuf[:], int64(HEADER_SIZE+int(obj.jointsCount)*0x10+i*0x18))
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

	var matdata [0x30]byte
	_, err = rdr.ReadAt(matdata[:], int64(obj.matOffset))
	if err != nil {
		return nil, err
	}

	obj.jointsCount2 = binary.LittleEndian.Uint32(matdata[0:4])
	if obj.jointsCount != obj.jointsCount2 {
		return nil, fmt.Errorf("Joints count not same in metdata and header (%.2x vs %.2x)",
			obj.jointsCount, obj.jointsCount2)
	}

	obj.unk1arrOffset = binary.LittleEndian.Uint32(matdata[4:8])
	obj.unk2arrOffset = binary.LittleEndian.Uint32(matdata[12:16])
	obj.unk3arrOffset = binary.LittleEndian.Uint32(matdata[32:36])
	obj.unk4arrOffset = binary.LittleEndian.Uint32(matdata[36:40])
	obj.unk5arrOffset = binary.LittleEndian.Uint32(matdata[40:44])
	obj.unk6arrOffset = binary.LittleEndian.Uint32(matdata[44:48])

	for _, j := range obj.Joints {
		var matrixBuffer [0x40]byte
		_, err = rdr.ReadAt(matrixBuffer[:], int64(obj.matOffset+0x30))
		if err != nil {
			return nil, err
		}
		for z := range j.Matrix {
			j.Matrix[z] = math.Float32frombits(binary.LittleEndian.Uint32(matrixBuffer[z*4 : z*4+4]))
		}

		var floatBuffer [0x10]byte
		_, err = rdr.ReadAt(floatBuffer[:], int64(obj.matOffset+obj.unk1arrOffset))
		if err != nil {
			return nil, err
		}
		for k := range j.Arr1 {
			j.Arr1[k] = math.Float32frombits(binary.LittleEndian.Uint32(floatBuffer[k*4 : k*4+4]))
		}

		_, err = rdr.ReadAt(floatBuffer[:], int64(obj.matOffset+obj.unk2arrOffset))
		if err != nil {
			return nil, err
		}
		for k := range j.Arr2 {
			j.Arr2[k] = math.Float32frombits(binary.LittleEndian.Uint32(floatBuffer[k*4 : k*4+4]))
		}

		_, err = rdr.ReadAt(floatBuffer[:], int64(obj.matOffset+obj.unk3arrOffset))
		if err != nil {
			return nil, err
		}
		for k := range j.Arr3 {
			j.Arr3[k] = math.Float32frombits(binary.LittleEndian.Uint32(floatBuffer[k*4 : k*4+4]))
		}

		_, err = rdr.ReadAt(floatBuffer[:], int64(obj.matOffset+obj.unk4arrOffset))
		if err != nil {
			return nil, err
		}
		for k := range j.Arr4 {
			j.Arr4[k] = math.Float32frombits(binary.LittleEndian.Uint32(floatBuffer[k*4 : k*4+4]))
		}

		_, err = rdr.ReadAt(floatBuffer[:], int64(obj.matOffset+obj.unk5arrOffset))
		if err != nil {
			return nil, err
		}
		for k := range j.Arr5 {
			j.Arr5[k] = math.Float32frombits(binary.LittleEndian.Uint32(floatBuffer[k*4 : k*4+4]))
		}

		_, err = rdr.ReadAt(floatBuffer[:], int64(obj.matOffset+obj.unk6arrOffset))
		if err != nil {
			return nil, err
		}
		for k := range j.Arr6 {
			j.Arr6[k] = math.Float32frombits(binary.LittleEndian.Uint32(floatBuffer[k*4 : k*4+4]))
		}

		if j.parent != 0xffff {
			obj.Joints[j.parent].SubJoints = append(obj.Joints[j.parent].SubJoints, j)
		}
	}

	for _, j := range obj.Joints {
		if j.parent == 0xffff {
			log.Printf("Joints tree:\n%s\n", j.String(""))
		}
	}

	return obj, nil
}

func (obj *Object) SaveJointsAsTree(outfname string) error {
	f, err := os.Create(outfname)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
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

	if err := obj.SaveJointsAsTree("logs/objtree-" + nd.Name + ".obj"); err != nil {
		return err
	}

	nd.Cache = obj
	return nil
}
