package mesh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"

	"github.com/mogaika/god_of_war_tools/files/mat"
	"github.com/mogaika/god_of_war_tools/files/wad"
)

type stUV struct {
	u, v float32
}

type stNorm struct {
	x, y, z float32
}

type stRGBA struct {
	r, g, b, a uint8
}

type stXYZ struct {
	x, y, z float32
	skip    bool
}

type stBlock struct {
	uvs   []stUV
	trias []stXYZ
	norms []stNorm
	blend []stRGBA
}

// GS use 12:4 fixed point format
// 1 << 12 = 4096
const GSFixed12Point4Delimeter = 4096.0

const MESH_MAGIC = 0x1000f

func init() {
	wad.PregisterExporter(MESH_MAGIC, &Mesh{})
}

func VifRead1(vif []byte, debug_off uint32) (error, []*stBlock) {
	result := make([]*stBlock, 0)

	currentblock := new(stBlock)

	u32 := func(idx uint32) uint32 {
		return binary.LittleEndian.Uint32(vif[idx : idx+4])
	}
	u16 := func(idx uint32) uint16 {
		return binary.LittleEndian.Uint16(vif[idx : idx+2])
	}
	u8 := func(idx uint32) uint8 {
		return vif[idx]
	}

	pos := uint32(0)

	spaces := "     "

	exit := false

	cl := 3

	for i := 0; !exit; i++ {
		pos = ((pos + 3) / 4) * 4
		if pos > uint32(len(vif)-4) {
			break
		}

		pk_cmd := u8(pos + 3)
		pk_num := u8(pos + 2)
		pk_dat1 := u8(pos)
		pk_dat2 := u8(pos + 1)

		pos += 4

		if pk_cmd >= 0x60 {
			components := ((pk_cmd >> 2) & 0x3) + 1
			bwidth := pk_cmd & 0x3
			widthmap := []uint32{32, 16, 8, 4} // 4 - rgb5a1 (only with 4 components
			width := widthmap[bwidth]

			blocksize := uint32(components) * ((width * uint32(pk_num)) / 8)

			signed := ((pk_dat2 & (1 << 6)) >> 6) ^ 1
			address := (pk_dat2 & (1 << 7)) >> 7

			target := uint32(pk_dat1) | (uint32(pk_dat2&3) << 8)

			/*
				struct VIFCodeUnpack
				{
				    unsigned short addr : 10; // адрес назначения (разделен на 16)
				    unsigned short unused : 4; // не используется
				    unsigned short usn : 1; // знак чисел: 0 - знаковое, 1 - беззнаковое.
				    unsigned short flg : 1; // флаг адресации данных ?.
				    unsigned char num; // количество данных для пересылки (обычно равно кол-ву вершин в меше)
				    unsigned char format: 4; // Тип формата данных (см. ниже)
				    unsigned char cmd : 4; // команда этого кода (UNPACK) - 6
				};
			*/

			grabbedType := "none"

			if cl == 3 || cl == 4 {
				if pk_cmd == 0x6d && components == 4 && width == 16 && signed != 0 {
					if currentblock.trias == nil {
						currentblock.trias = make([]stXYZ, 0)
					}

					bp := pos
					for i := uint8(0); i < pk_num; i++ {

						x := float32(int16(u16(bp))) / GSFixed12Point4Delimeter
						y := float32(int16(u16(bp+2))) / GSFixed12Point4Delimeter
						z := float32(int16(u16(bp+4))) / GSFixed12Point4Delimeter
						skip := u8(bp+7)&0x80 != 0

						currentblock.trias = append(currentblock.trias, stXYZ{x: x, y: y, z: z, skip: skip})

						// log.Printf(" -- %.4x %+2.4f %+2.4f %+2.4f", u16(bp+6), x, y, z)

						bp += 8
					}

					grabbedType = " xyz"
				} else if pk_cmd == 0x6e && components == 4 && width == 8 && signed == 0 {
					if currentblock.blend == nil {
						currentblock.blend = make([]stRGBA, 0)
					}
					bp := pos
					for i := uint8(0); i < pk_num; i++ {
						currentblock.blend = append(currentblock.blend,
							stRGBA{r: u8(bp), g: u8(bp + 1), b: u8(bp + 2), a: u8(bp + 3)})

						bp += 4
					}

					grabbedType = "rgba"
				} else if pk_cmd == 0x65 && components == 2 && width == 16 && signed == 1 {
					if currentblock.uvs == nil {
						currentblock.uvs = make([]stUV, 0)
					}
					bp := pos
					for i := uint8(0); i < pk_num; i++ {
						currentblock.uvs = append(currentblock.uvs,
							stUV{u: float32(int16(u16(bp))) / GSFixed12Point4Delimeter,
								v: float32(int16(u16(bp+2))) / GSFixed12Point4Delimeter})

						bp += 4
					}

					grabbedType = " uv "
				}
			} else {
				if components == 4 && width == 32 {
					for i := 0; i < int(pk_num); i++ {
						log.Printf("    [%.2d]%.8x %.8x %.8x %.8x", i, u32(pos), u32(pos+4), u32(pos+8), u32(pos+12))
						log.Printf("          %f %f %f %f",
							math.Float32frombits(u32(pos)), math.Float32frombits(u32(pos+4)),
							math.Float32frombits(u32(pos+8)), math.Float32frombits(u32(pos+12)))
					}
				}

			}
			log.Printf("%s %.6x vif unpack: [%s] %.2x elements: %.2x components: %d type: %.2d target: %.3x sign: %d addr: %d size: %.4x",
				spaces, debug_off+pos, grabbedType, pk_cmd, pk_num, components, width, target, signed, address, blocksize)

			pos += blocksize
		} else {
			switch pk_cmd {
			case 0:
				// log.Printf("%s %.6x nop", spaces, debug_off+pos)
			case 01:
				log.Printf("%s %.6x Stcycl wl=%.2x cl=%.2x", spaces, debug_off+pos, pk_dat2, pk_dat1)
				cl = int(pk_dat1)
				if cl == 1 {
					if currentblock.trias != nil && len(currentblock.trias) > 0 {
						result = append(result, currentblock)
					}
					currentblock = new(stBlock)
				}
			case 05:
				// cmode := " pos "
				/*
							enum // Decompression modes
					case 0:
							{
								Normal = 0,
								OffsetDecompression, // would conflict with vif code
								Difference
							}
				*/
				// switch pk_dat1 {
				// case 1:
				// 	cmode = "[pos]"
				// case 2:
				// 	cmode = "[cur]"
				// }
				// log.Printf("%s %.6x Stmod  mode=%s (%d)", spaces, debug_off+pos, cmode, pk_dat1)
			case 0x14:
				log.Printf("%s %.6x Mscall proc command", spaces, debug_off+pos)
			case 0x30:
				log.Printf("%s %.6x Strow  proc command", spaces, debug_off+pos)
			default:
				log.Printf("%s %.6x VIF command: %.2x:%.2x data: %.2x:%.2x", spaces, debug_off+pos, pk_cmd, pk_num, pk_dat1, pk_dat2)
				exit = true
			}
		}

	}
	return nil, result
}

type MeshPacket struct {
	fileStruct uint32
	Blocks     []stBlock
}

type MeshObject struct {
	fileStruct uint32
	Type       uint16
	TextureId  uint8
	Packets    []MeshPacket
}

type MeshGroup struct {
	fileStruct uint32
	Objects    []MeshObject
}

type MeshPart struct {
	fileStruct uint32
	Groups     []MeshGroup
}

type Mesh struct {
	CommentStart uint32
	Parts        []MeshPart
	File         []byte
}

func NewFromData(rdat io.Reader) (*Mesh, error) {
	file, err := ioutil.ReadAll(rdat)
	if err != nil {
		return nil, err
	}

	u32 := func(idx uint32) uint32 {
		return binary.LittleEndian.Uint32(file[idx : idx+4])
	}
	u16 := func(idx uint32) uint16 {
		return binary.LittleEndian.Uint16(file[idx : idx+2])
	}
	u8 := func(idx uint32) uint8 {
		return file[idx]
	}

	if u32(0) != MESH_MAGIC {
		return nil, fmt.Errorf("Unknown mesh type")
	}

	mdlCommentStart := u32(4)
	if mdlCommentStart > uint32(len(file)) {
		mdlCommentStart = uint32(len(file))
	}

	partsCount := u32(8)

	parts := make([]MeshPart, partsCount)

	//log.Printf("parts: %d", partsCount)

	// build tree for blocks boundary finding
	for iPart := uint32(0); iPart < partsCount; iPart++ {
		pPart := u32(0x50 + iPart*4)
		groupsCount := uint32(u16(pPart + 2))

		parts[iPart].fileStruct = pPart
		groups := make([]MeshGroup, groupsCount)

		for iGroup := uint32(0); iGroup < groupsCount; iGroup++ {
			pGroup := pPart + u32(pPart+iGroup*4+4)
			objectsCount := u32(pGroup + 4)

			groups[iGroup].fileStruct = pGroup
			objects := make([]MeshObject, objectsCount)
			for iObject := uint32(0); iObject < objectsCount; iObject++ {
				pObject := pGroup + u32(0xc+pGroup+iObject*4)
				tObject := u16(pObject)
				packetsCount := u32(pObject+0xc) * uint32(u8(pObject+0x18))

				objects[iObject].fileStruct = pObject
				objects[iObject].Type = tObject
				packets := make([]MeshPacket, packetsCount)

				/*
					0x1d - static mesh (bridge, skybox)
					0x0e - dynamic? mesh (ship, hero, enemy)
				*/

				if tObject == 0xe || tObject == 0x1d || tObject == 0x24 {
					objects[iObject].TextureId = u8(pObject + 8)

					//log.Printf("%.6x : texture %.2d : %.8x %.8x %.8x %.8x %.8x %.8x %.8x %.8x", pObject, objects[iObject].TextureId,
					//	u32(pObject), u32(pObject+4), u32(pObject+8), u32(pObject+12),
					//	u32(pObject+16), u32(pObject+20), u32(pObject+24), u32(pObject+28))

					for iPacket := uint32(0); iPacket < packetsCount; iPacket++ {
						pPacketInfo := pObject + 0x20 + iPacket*0x10
						pPacket := pObject + u32(pPacketInfo+4)

						packets[iPacket].fileStruct = pPacket
					}
				}
				objects[iObject].Packets = packets
			}
			groups[iGroup].Objects = objects
		}
		parts[iPart].Groups = groups
	}

	mesh := &Mesh{CommentStart: mdlCommentStart,
		Parts: parts,
		File:  file}

	return mesh, nil
}

func (ms *Mesh) Extract(textures []string, outfname string) ([]string, error) {
	ofileName := outfname + ".obj"

	err := os.MkdirAll(path.Dir(ofileName), 0777)
	if err != nil {
		return nil, err
	}

	ofile, err := os.Create(ofileName)
	if err != nil {
		log.Fatalf("Cannot create file %s: %v", ofileName, err)
	}
	defer ofile.Close()
	vertIndex := 1
	textIndex := 1

	oMtlFileName := outfname + ".mtl"
	_, oMtlRelativeName := path.Split(oMtlFileName)
	omtlFile, err := os.Create(oMtlFileName)
	if err != nil {
		return nil, err
	}

	for i, tex := range textures {
		fmt.Fprintf(omtlFile, "newmtl mat_%d\n", i)
		fmt.Fprintf(omtlFile, "Ka 1.000 1.000 1.000\nKd 1.000 1.000 1.000\nKs 0.000 0.000 0.000\n")
		if tex != "" {
			fmt.Fprintf(omtlFile, "d 1.000000\n") // for transparent textures parts
			fmt.Fprintf(omtlFile, "map_Ka %s\nmap_Kd %s\n\n", tex, tex)
		}
	}
	omtlFile.Close()

	fmt.Fprintf(ofile, "mtllib %s\n\n", oMtlRelativeName)

	pointerEnd := ms.CommentStart
	parts := ms.Parts
	for iPart := len(parts) - 1; iPart >= 0; iPart-- {
		part := &parts[iPart]
		groups := part.Groups

		log.Printf(" part: %d pos: %.6x; groups: %d", iPart, part.fileStruct, len(groups))
		fmt.Fprintf(ofile, "g group_%.6x\n", part.fileStruct)

		for iGroup := len(groups) - 1; iGroup >= 0; iGroup-- {
			group := &groups[iGroup]
			objects := group.Objects

			for iObject := len(objects) - 1; iObject >= 0; iObject-- {
				object := &objects[iObject]
				packets := object.Packets

				log.Printf("   object: %d pos: %.6x; type: %.2x; textureid: %.2x", iObject, object.fileStruct, object.Type, object.TextureId)
				swp := true

				bufv := ""
				bufvt := ""
				buff := ""

				for iPacket := len(packets) - 1; iPacket >= 0; iPacket-- {
					packet := &packets[iPacket]

					log.Printf("    packet: %d pos: %.6x;", iPacket, packet.fileStruct)
					if packet.fileStruct >= pointerEnd {
						break
					}
					err, vifmeshs := VifRead1(ms.File[packet.fileStruct:pointerEnd], packet.fileStruct)
					if err != nil {
						return nil, err
					} else {
						for _, mesh := range vifmeshs {
							uv := mesh.uvs != nil && len(mesh.uvs) == len(mesh.trias)

							for i := range mesh.trias {
								t := &mesh.trias[i]

								bufv += fmt.Sprintf("v %f %f %f\n", t.x, t.y, t.z)
								if uv {
									tx := &mesh.uvs[i]
									bufvt += fmt.Sprintf("vt %f %f\n", tx.u, 1.0-tx.v)
								}

								if !t.skip {
									i2 := 1
									i3 := 2
									if swp {
										i2, i3 = i3, i2
									}

									if uv {
										buff += fmt.Sprintf("f %d/%d %d/%d %d/%d\n",
											vertIndex, textIndex, vertIndex-i2, textIndex-i2, vertIndex-i3, textIndex-i3)
									} else {
										buff += fmt.Sprintf("f %d %d %d\n", vertIndex, vertIndex-i2, vertIndex-i3)
									}
								}
								swp = !swp

								vertIndex++
								if uv {
									textIndex++
								}
							}
							fmt.Fprintf(ofile, "\n\n")
						}
					}
					pointerEnd = packet.fileStruct
				}

				fmt.Fprintf(ofile, "o obj_%.6x\n", object.fileStruct)
				ofile.WriteString(bufv)
				ofile.WriteString(bufvt)
				fmt.Fprintf(ofile, "usemtl mat_%d\n", object.TextureId)
				ofile.WriteString(buff)

				pointerEnd = object.fileStruct
			}
			pointerEnd = group.fileStruct
		}
		pointerEnd = part.fileStruct
	}

	return []string{ofileName}, nil
}

func (*Mesh) ExtractFromNode(nd *wad.WadNode, outfname string) error {
	log.Printf("Mesh '%s' extraction", nd.Name)

	pathPrefix := "../"
	for i := 0; i < nd.Depth; i++ {
		pathPrefix += "../"
	}

	var textures []string
	for _, v := range nd.Parent.SubNodes {
		if v.Type == wad.NODE_TYPE_LINK {
			v = v.LinkTo
		}
		if v.Format == mat.MAT_MAGIC {
			if !v.Extracted || v.Cache == nil {
				return errors.New("Material not loaded before mesh")
			} else {
				mat := v.Cache.(*mat.Material)
				if mat == nil || mat.Layers == nil || len(mat.Layers) == 0 {
					return fmt.Errorf("Material '%s' not cached ", v.Path)
				}

				if mat.Layers[0].Texture != "" {
					t := nd.Find(mat.Layers[0].Texture, true)
					if !t.Extracted || t.ExtractedNames == nil || len(t.ExtractedNames) == 0 {
						return errors.New("Material not loaded before mesh")
					} else {
						tex := t.ExtractedNames[0]
						texPath := path.Join(pathPrefix, tex)
						textures = append(textures, texPath)
					}
				} else {
					log.Printf("Mat without texture '%s'", v.Name)
					textures = append(textures, "")
				}
			}
		}
	}

	reader, err := nd.DataReader()
	if err != nil {
		return err
	}

	mesh, err := NewFromData(reader)
	if err != nil {
		return err
	}

	resNames, err := mesh.Extract(textures, outfname)
	if err != nil {
		return err
	}

	nd.ExtractedNames = resNames
	nd.Cache = mesh
	return nil
}
