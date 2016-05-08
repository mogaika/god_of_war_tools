package obj

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/go-gl/mathgl/mgl32"

	"github.com/mogaika/god_of_war_tools/files/wad"
	"github.com/mogaika/god_of_war_tools/utils"
)

const OBJECT_MAGIC = 0x00040001
const HEADER_SIZE = 0x2C
const DATA_HEADER_SIZE = 0x30

type Joint struct {
	Id          uint16
	Name        string
	ChildsStart uint16
	ChildsEnd   uint16
	Parent      uint16

	HaveInverse bool
	InvId       uint16
}

const JOINT_CHILD_NONE = 0xffff

type Object struct {
	Joints []Joint

	dataOffset  uint32
	jointsCount uint32

	Mat1count  uint32
	Vec2offset uint32 // or this is end of matrixes1
	Vec2count  uint32 // maybe zero = 1,1=2,2=3,...
	Mat3offset uint32
	Mat3count  uint32
	Vec4offset uint32
	Vec5offset uint32
	Vec6offset uint32
	Vec7offset uint32

	Matrixes1 []mgl32.Mat4 // bind pose
	Vectors2  [][4]uint32
	Matrixes3 []mgl32.Mat4 // inverce matrices (not at all joints)
	Vectors4  []mgl32.Vec4 // bind pose xyz
	Vectors5  [][4]int32
	Vectors6  []mgl32.Vec4 // bind pose scale
	Vectors7  []mgl32.Vec4
}

func init() {
	wad.PregisterExporter(OBJECT_MAGIC, &Object{})
}

func (obj *Object) StringJoint(id uint16, spaces string) string {
	j := obj.Joints[id]
	return fmt.Sprintf("%sjoint [%.4x <=%.4x %.4x->%.4x %t:%.4x]  %s:\n%srot: %#v\n%spos: %#v\n%sv5 : %#v\n%ssiz: %#v\n%sv7 : %#v\n",
		spaces, j.Id, j.Parent, j.ChildsStart, j.ChildsEnd, j.HaveInverse, j.InvId, j.Name,
		spaces, obj.Matrixes1[j.Id], spaces, obj.Vectors4[j.Id],
		spaces, obj.Vectors5[j.Id], spaces, obj.Vectors6[j.Id],
		spaces, obj.Vectors7[j.Id])
}

func (obj *Object) StringTree() string {
	stack := make([]uint16, 0, 32)
	spaces := string(make([]byte, 0, 64))
	spaces = ""

	var buffer bytes.Buffer

	for i := uint16(0); i < uint16(obj.jointsCount); i++ {
		j := obj.Joints[i]

		if j.Parent != JOINT_CHILD_NONE {
			for i == stack[len(stack)-1] {
				stack = stack[:len(stack)-1]
				spaces = spaces[:len(spaces)-2]
			}
		}

		buffer.WriteString(obj.StringJoint(i, spaces))

		if j.ChildsStart != JOINT_CHILD_NONE {
			if j.ChildsEnd == uint16(0xffff) && len(stack) > 0 {
				stack = append(stack, stack[len(stack)-1])
			} else {
				stack = append(stack, j.ChildsEnd)
			}
			spaces += "  "
		}
	}
	return buffer.String()
}

func NewFromData(rdr io.ReaderAt) (*Object, error) {
	var file [HEADER_SIZE]byte
	_, err := rdr.ReadAt(file[:], 0)
	if err != nil {
		return nil, err
	}

	obj := new(Object)

	log.Printf(" OBJ: %.8x %.8x %.8x   %.8x %.8x %.8x",
		binary.LittleEndian.Uint32(file[0x4:0x8]),
		binary.LittleEndian.Uint32(file[0x8:0xc]),
		binary.LittleEndian.Uint32(file[0xc:0x10]),
		binary.LittleEndian.Uint32(file[0x10:0x14]),
		binary.LittleEndian.Uint32(file[0x14:0x18]),
		binary.LittleEndian.Uint32(file[0x18:0x1c]))

	obj.jointsCount = binary.LittleEndian.Uint32(file[0x1c:0x20])
	obj.dataOffset = binary.LittleEndian.Uint32(file[0x28:0x2c])

	obj.Joints = make([]Joint, obj.jointsCount)

	invid := uint16(0)
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

		flags := binary.LittleEndian.Uint32(jointBuf[0:4])

		isInvMat := flags&0xa0 == 0xa0
		obj.Joints[i] = Joint{
			Name:        utils.BytesToString(nameBuf[:]),
			ChildsStart: binary.LittleEndian.Uint16(jointBuf[0x4:0x6]),
			ChildsEnd:   binary.LittleEndian.Uint16(jointBuf[0x6:0x8]),
			Parent:      binary.LittleEndian.Uint16(jointBuf[0x8:0xa]),
			Id:          uint16(i),
			HaveInverse: isInvMat,
			InvId:       invid,
		}

		if isInvMat {
			invid++
		}
	}

	var matdata [DATA_HEADER_SIZE]byte
	_, err = rdr.ReadAt(matdata[:], int64(obj.dataOffset))
	if err != nil {
		return nil, err
	}

	obj.Mat1count = binary.LittleEndian.Uint32(matdata[0:4])
	obj.Vec2offset = binary.LittleEndian.Uint32(matdata[4:8])
	obj.Vec2count = binary.LittleEndian.Uint32(matdata[8:12])
	obj.Mat3offset = binary.LittleEndian.Uint32(matdata[12:16])
	obj.Mat3count = binary.LittleEndian.Uint32(matdata[16:20])
	obj.Vec4offset = binary.LittleEndian.Uint32(matdata[32:36])
	obj.Vec5offset = binary.LittleEndian.Uint32(matdata[36:40])
	obj.Vec6offset = binary.LittleEndian.Uint32(matdata[40:44])
	obj.Vec7offset = binary.LittleEndian.Uint32(matdata[44:48])

	obj.Matrixes1 = make([]mgl32.Mat4, obj.Mat1count)
	obj.Vectors2 = make([][4]uint32, obj.Vec2count+1)
	obj.Matrixes3 = make([]mgl32.Mat4, obj.Mat3count)
	obj.Vectors4 = make([]mgl32.Vec4, obj.Mat1count)
	obj.Vectors5 = make([][4]int32, obj.Mat1count)
	obj.Vectors6 = make([]mgl32.Vec4, obj.Mat1count)
	obj.Vectors7 = make([]mgl32.Vec4, obj.Mat1count)

	mat1buf := make([]byte, len(obj.Matrixes1)*0x40)
	vec2buf := make([]byte, len(obj.Vectors2)*0x10)
	mat3buf := make([]byte, len(obj.Matrixes3)*0x40)
	vec4buf := make([]byte, len(obj.Vectors4)*0x10)
	vec5buf := make([]byte, len(obj.Vectors5)*0x10)
	vec6buf := make([]byte, len(obj.Vectors6)*0x10)
	vec7buf := make([]byte, len(obj.Vectors7)*0x10)

	if _, err = rdr.ReadAt(mat1buf[:], int64(obj.dataOffset+DATA_HEADER_SIZE)); err != nil {
		return nil, err
	}
	if _, err = rdr.ReadAt(vec2buf[:], int64(obj.dataOffset+obj.Vec2offset)); err != nil {
		return nil, err
	}
	if _, err = rdr.ReadAt(mat3buf[:], int64(obj.dataOffset+obj.Mat3offset)); err != nil {
		return nil, err
	}
	if _, err = rdr.ReadAt(vec4buf[:], int64(obj.dataOffset+obj.Vec4offset)); err != nil {
		return nil, err
	}
	if _, err = rdr.ReadAt(vec5buf[:], int64(obj.dataOffset+obj.Vec5offset)); err != nil {
		return nil, err
	}
	if _, err = rdr.ReadAt(vec6buf[:], int64(obj.dataOffset+obj.Vec6offset)); err != nil {
		return nil, err
	}
	if _, err = rdr.ReadAt(vec7buf[:], int64(obj.dataOffset+obj.Vec7offset)); err != nil {
		return nil, err
	}

	for i := range obj.Matrixes1 {
		obj.Matrixes1[i] = utils.BytesToMat4f(mat1buf[i*0x40 : i*0x40+0x40])
	}
	for i := range obj.Vectors2 {
		obj.Vectors2[i] = utils.BytesToVec4u(vec2buf[i*0x10 : i*0x10+0x10])
	}
	for i := range obj.Matrixes3 {
		obj.Matrixes3[i] = utils.BytesToMat4f(mat3buf[i*0x40 : i*0x40+0x40])
	}
	for i := range obj.Vectors4 {
		obj.Vectors4[i] = utils.BytesToVec4f(vec4buf[i*0x10 : i*0x10+0x10])
		obj.Vectors5[i] = utils.BytesToVec4i(vec5buf[i*0x10 : i*0x10+0x10])
		obj.Vectors6[i] = utils.BytesToVec4f(vec6buf[i*0x10 : i*0x10+0x10])
		obj.Vectors7[i] = utils.BytesToVec4f(vec7buf[i*0x10 : i*0x10+0x10])
	}

	s := ""
	for _, m := range obj.Matrixes3 {
		s += fmt.Sprintf("\n   m3: %#v", m)
	}
	/*for _, j := range obj.Joints {
		s += fmt.Sprintf("\njoint [%.4x <=%.4x %.4x->%.4x]  %s:\nm1: %#v\nv4: %#v\nv5: %#v\nv6: %#v\nv7: %#v",
			j.Id, j.Parent, j.ChildsStart, j.ChildsEnd, j.Name, obj.Matrixes1[j.Id],
			obj.Vectors4[j.Id], obj.Vectors5[j.Id], obj.Vectors6[j.Id], obj.Vectors7[j.Id])
	}*/
	log.Printf("%s\n%s", s, obj.StringTree())

	return obj, nil
}

func (obj *Object) SaveJointsAsTree(outfname string) error {
	f, err := os.Create(outfname)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, j := range obj.Joints {

		local_pos := obj.Vectors4[i].Vec3()
		for p := j.Parent; p != JOINT_CHILD_NONE; p = obj.Joints[p].Parent {

			local_pos = mgl32.TransformCoordinate(local_pos, obj.Matrixes1[p])

			//local_pos = local_pos.Add(obj.Vectors4[p].Vec3())
			//local_pos = mgl32.Mat4ToQuat(obj.Matrixes1[p]).Rotate(local_pos)
		}

		log.Printf("%.4x = %f %f %f", i, local_pos.X(), local_pos.Y(), local_pos.Z())
		fmt.Fprintf(f, "v %f %f %f\n", local_pos.X(), local_pos.Y(), local_pos.Z())
	}

	for i, j := range obj.Joints {
		if j.Parent != JOINT_CHILD_NONE {
			fmt.Fprintf(f, "f %d %d\n", i+1, j.Parent+1)
		}
	}

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
