package mesh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	uvs      []stUV
	trias    []stXYZ
	norms    []stNorm
	blend    []stRGBA
	debugPos uint32
}

type MeshPacket struct {
	fileStruct uint32
	Rows       uint16
	Blocks     []*stBlock
}

type MeshObject struct {
	fileStruct uint32
	Type       uint16
	MaterialId uint8
	Packets    []*MeshPacket
}

type MeshGroup struct {
	fileStruct uint32
	Objects    []*MeshObject
}

type MeshPart struct {
	fileStruct uint32
	Groups     []*MeshGroup
}

type Mesh struct {
	CommentStart uint32
	Parts        []*MeshPart
	File         []byte
}

// GS use 12:4 fixed point format
// 1 << 12 = 4096
const GSFixed12Point4Delimeter = 4096.0

const MESH_MAGIC = 0x1000f

func init() {
	wad.PregisterExporter(MESH_MAGIC, &Mesh{})
}

func VifRead1(vif []byte, debug_off uint32) (error, []*stBlock) {
	/*
		What game send on vif:

		Stcycl wl=1 cl=1/2/3/4

		One of array:
		[ xyzw4_16i ] -
			only position (GUI)
		[ rgba4_08u , xyzw4_16i ] -
			color and position (GUI + Effects)
		[   uv2_16i , xyzw4_16i ] -
			texture coords and position (simple model)
		[   uv2_16i , rgba4_08u , xyzw4_16i ] -
			texture coords + blend for hand-shaded models + position
		[   uv2_16i , norm2_16i , rgba4_08u , xyzw4_16i ] -
			texture coords + normal vector + blend color + position for hard materials

		Stcycl wl=1 cl=1

		Command vars:
		[ xyzw4_32i ] -
			paket refrence (verticles count, joint assign, joint types).
			used stable targets: 000, 155, 2ab
		[ xyzw4_32i ] -
			material refrence ? (diffuse/ambient colors, alpha)?

		Mscall (if not last sequence in packet) - process data

		Anyway position all time xyzw4_16i and last in sequence
	*/

	result := make([]*stBlock, 0)

	var block_data_xyzw []byte = nil
	var block_data_rgba []byte = nil
	var block_data_uv []byte = nil
	block_data_uv_width := 0
	var block_data_norm []byte = nil

	pos := uint32(0)
	spaces := "     "
	exit := false
	flush := false

	for iCommandInBlock := 0; !exit; iCommandInBlock++ {
		pos = ((pos + 3) / 4) * 4
		if pos >= uint32(len(vif)) {
			break
		}

		pk_cmd := vif[pos+3]
		pk_num := vif[pos+2]
		pk_dat2 := vif[pos+1]
		pk_dat1 := vif[pos]

		tagpos := pos
		pos += 4

		if pk_cmd >= 0x60 { // if unpack command
			components := ((pk_cmd >> 2) & 0x3) + 1
			bwidth := pk_cmd & 0x3
			widthmap := []uint32{32, 16, 8, 4} // 4 = r5g5b5a1
			width := widthmap[bwidth]

			blocksize := uint32(components) * ((width * uint32(pk_num)) / 8)

			signed := ((pk_dat2&(1<<6))>>6)^1 != 0
			address := (pk_dat2&(1<<7))>>7 != 0

			target := uint32(pk_dat1) | (uint32(pk_dat2&3) << 8)

			handledBy := ""

			switch width {
			case 32:
				if signed {
					switch components {
					case 4: // joints and format info all time after data (i think)
						flush = true
						handledBy = "meta"
						for i := byte(0); i < pk_num; i++ {
							bp := pos + uint32(i*0x10)
							log.Printf("%s -  %.6x = %.4x %.4x %.4x %.4x", spaces, bp,
								binary.LittleEndian.Uint16(vif[bp:bp+2]), binary.LittleEndian.Uint16(vif[bp+2:bp+4]),
								binary.LittleEndian.Uint16(vif[bp+4:bp+6]), binary.LittleEndian.Uint16(vif[bp+6:bp+8]))
						}
					case 2:
						handledBy = " uv4"
						if block_data_uv == nil {
							block_data_uv = vif[pos : pos+blocksize]
							handledBy = " uv2"
							block_data_uv_width = 4
						} else {
							return fmt.Errorf("UV already present. What is this: %.6x ?", tagpos+debug_off), nil
						}
					}
				}
			case 16:
				if signed {
					switch components {
					case 4:
						if block_data_xyzw == nil {
							block_data_xyzw = vif[pos : pos+blocksize]
							handledBy = "xyzw"
						} else {
							return fmt.Errorf("XYZW already present. What is this: %.6x ?", tagpos+debug_off), nil
						}
					case 2:
						if block_data_uv == nil {
							block_data_uv = vif[pos : pos+blocksize]
							handledBy = " uv2"
							block_data_uv_width = 2
						} else {
							return fmt.Errorf("UV already present. What is this: %.6x ?", tagpos+debug_off), nil
						}
					}
				}
			case 8:
				if signed {
					switch components {
					case 3:
						if block_data_norm == nil {
							block_data_norm = vif[pos : pos+blocksize]
							handledBy = "norm"
						} else {
							return fmt.Errorf("NORM already present. What is this: %.6x ?", tagpos+debug_off), nil
						}
					}
				} else {
					switch components {
					case 4:
						if block_data_rgba == nil {
							block_data_rgba = vif[pos : pos+blocksize]
							handledBy = "rgba"
						} else {
							return fmt.Errorf("RGBA already present. What is this: %.6x ?", tagpos+debug_off), nil
						}
					}
				}
			}

			if handledBy == "" {
				return fmt.Errorf("Block %.6x (cmd %.2x; %d bit; %d components; %d elements; sign %t; addr %t; target: %.3x; size: %.6x) not handled",
					tagpos+debug_off, pk_cmd, width, components, pk_num, signed, address, target, blocksize), nil
			} else {
				log.Printf("%s %.6x vif unpack [%s]: %.2x elements: %.2x components: %d type: %.2d target: %.3x sign: %t addr: %t size: %.6x",
					spaces, debug_off+tagpos, handledBy, pk_cmd, pk_num, components, width, target, signed, address, blocksize)
			}

			pos += blocksize
		} else {
			switch pk_cmd {
			case 0:
				log.Printf("%s %.6x nop", spaces, debug_off+tagpos)
			case 01:
				log.Printf("%s %.6x Stcycl wl=%.2x cl=%.2x", spaces, debug_off+tagpos, pk_dat2, pk_dat1)
			case 05:
				cmode := " pos "
				/*	 Decompression modes
				Normal = 0,
				OffsetDecompression, // would conflict with vif code
				Difference
				*/
				switch pk_dat1 {
				case 1:
					cmode = "[pos]"
				case 2:
					cmode = "[cur]"
				}
				log.Printf("%s %.6x Stmod  mode=%s (%d)", spaces, debug_off+tagpos, cmode, pk_dat1)
			case 0x14:
				log.Printf("%s %.6x Mscall proc command", spaces, debug_off+tagpos)
				flush = true
			case 0x30:
				log.Printf("%s %.6x Strow  proc command", spaces, debug_off+tagpos)
				pos += 0x10
			default:
				return fmt.Errorf("Unknown %.6x VIF command: %.2x:%.2x data: %.2x:%.2x",
					debug_off+tagpos, pk_cmd, pk_num, pk_dat1, pk_dat2), nil
			}
		}

		if flush || exit {
			flush = false

			// if we collect some data
			if block_data_xyzw != nil {
				currentBlock := &stBlock{}
				currentBlock.debugPos = tagpos

				currentBlock.trias = make([]stXYZ, len(block_data_xyzw)/8)
				for i := range currentBlock.trias {
					bp := i * 8
					t := &currentBlock.trias[i]
					t.x = float32(int16(binary.LittleEndian.Uint16(block_data_xyzw[bp:bp+2]))) / GSFixed12Point4Delimeter
					t.y = float32(int16(binary.LittleEndian.Uint16(block_data_xyzw[bp+2:bp+4]))) / GSFixed12Point4Delimeter
					t.z = float32(int16(binary.LittleEndian.Uint16(block_data_xyzw[bp+4:bp+6]))) / GSFixed12Point4Delimeter
					t.skip = block_data_xyzw[bp+7]&0x80 != 0
				}

				if block_data_uv != nil {
					switch block_data_uv_width {
					case 2:
						currentBlock.uvs = make([]stUV, len(block_data_uv)/4)
						for i := range currentBlock.trias {
							bp := i * 4
							u := &currentBlock.uvs[i]
							u.u = float32(int16(binary.LittleEndian.Uint16(block_data_uv[bp:bp+2]))) / GSFixed12Point4Delimeter
							u.v = float32(int16(binary.LittleEndian.Uint16(block_data_uv[bp+2:bp+4]))) / GSFixed12Point4Delimeter
						}
					case 4:
						currentBlock.uvs = make([]stUV, len(block_data_uv)/8)
						for i := range currentBlock.trias {
							bp := i * 8
							u := &currentBlock.uvs[i]
							u.u = float32(int32(binary.LittleEndian.Uint32(block_data_uv[bp:bp+4]))) / GSFixed12Point4Delimeter
							u.v = float32(int32(binary.LittleEndian.Uint32(block_data_uv[bp+4:bp+8]))) / GSFixed12Point4Delimeter
						}
					}
				}

				if block_data_norm != nil {
					currentBlock.norms = make([]stNorm, len(block_data_norm)/3)
					for i := range currentBlock.norms {
						bp := i * 3
						n := &currentBlock.norms[i]
						n.x = float32(int8(block_data_norm[bp])) / 100.0
						n.y = float32(int8(block_data_norm[bp+1])) / 100.0
						n.z = float32(int8(block_data_norm[bp+2])) / 100.0
					}
				}

				if block_data_rgba != nil {
					currentBlock.blend = make([]stRGBA, len(block_data_norm)/4)
					for i := range currentBlock.blend {
						bp := i * 4
						c := &currentBlock.blend[i]
						c.r = block_data_norm[bp]
						c.g = block_data_norm[bp+1]
						c.b = block_data_norm[bp+2]
						c.a = block_data_norm[bp+3]
					}
				}

				result = append(result, currentBlock)

				log.Printf("%s = Flush xyzw:%t, rgba:%t, uv:%t, norm:%t", spaces,
					block_data_xyzw != nil, block_data_rgba != nil,
					block_data_uv != nil, block_data_norm != nil)

				block_data_norm = nil
				block_data_rgba = nil
				block_data_xyzw = nil
				block_data_uv = nil

			}
		}
	}
	return nil, result
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
	parts := make([]*MeshPart, partsCount)
	for iPart := range parts {
		pPart := u32(0x50 + uint32(iPart)*4)
		groupsCount := u16(pPart + 2)

		part := &MeshPart{
			fileStruct: pPart,
			Groups:     make([]*MeshGroup, groupsCount),
		}
		parts[iPart] = part

		for iGroup := range part.Groups {
			pGroup := pPart + u32(pPart+uint32(iGroup)*4+4)
			objectsCount := u32(pGroup + 4)

			group := &MeshGroup{
				fileStruct: pGroup,
				Objects:    make([]*MeshObject, objectsCount),
			}

			part.Groups[iGroup] = group

			for iObject := range group.Objects {
				pObject := pGroup + u32(0xc+pGroup+uint32(iObject)*4)

				objectType := u16(pObject)
				packetsCount := u32(pObject+0xc) * uint32(u8(pObject+0x18))

				/*
					0x1d - static mesh (bridge, skybox)
					0x0e - dynamic? mesh (ship, hero, enemy)
				*/

				object := &MeshObject{
					fileStruct: pGroup,
					Type:       objectType,
					Packets:    make([]*MeshPacket, 0),
				}

				group.Objects[iObject] = object

				if objectType == 0xe || objectType == 0x1d || objectType == 0x24 {
					object.MaterialId = u8(pObject + 8)

					for iPacket := uint32(0); iPacket < packetsCount; iPacket++ {
						pPacketInfo := pObject + 0x20 + iPacket*0x10
						pPacket := pObject + u32(pPacketInfo+4)

						packet := &MeshPacket{
							fileStruct: pPacket,
							Rows:       u16(pPacketInfo),
						}

						object.Packets = append(object.Packets, packet)

						packetSize := uint32(packet.Rows) * 0x10
						packetEnd := packetSize + packet.fileStruct

						log.Printf("    packet: %d pos: %.6x rows: %.4x end: %.6x",
							iPacket, packet.fileStruct, packet.Rows, packetEnd)

						err, packet.Blocks = VifRead1(file[packet.fileStruct:packetEnd], packet.fileStruct)
						if err != nil {
							return nil, err
						}
					}
				}

			}
		}
	}

	mesh := &Mesh{CommentStart: mdlCommentStart,
		Parts: parts,
		File:  file}

	return mesh, nil
}

func (ms *Mesh) ExtractObj(textures []string, outfname string) ([]string, error) {
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
	normIndex := 1

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

	for iPart, part := range ms.Parts {
		log.Printf(" part: %d pos: %.6x; groups: %d", iPart, part.fileStruct, len(part.Groups))
		for iGroup, group := range part.Groups {
			log.Printf("  group: %d pos: %.6x; objects: %d", iGroup, group.fileStruct, len(group.Objects))
			for iObject, object := range group.Objects {
				log.Printf("   object: %d pos: %.6x; type: %.2x; materialid: %.2x", iObject, object.fileStruct, object.Type, object.MaterialId)
				for iPacket, packet := range object.Packets {
					packetSize := uint32(packet.Rows) * 0x10
					packetEnd := packetSize + packet.fileStruct

					log.Printf("    packet: %d pos: %.6x rows: %.4x end: %.6x",
						iPacket, packet.fileStruct, packet.Rows, packetEnd)

					err, vifmeshs := VifRead1(ms.File[packet.fileStruct:packetEnd], packet.fileStruct)
					if err != nil {
						return nil, err
					} else {
						for _, mesh := range vifmeshs {
							uv := mesh.uvs != nil && len(mesh.uvs) == len(mesh.trias)
							vn := mesh.norms != nil
							if vn && len(mesh.norms) != len(mesh.trias) {
								vn = false
								log.Printf("Norm not match verts : %d vs %d", len(mesh.norms), len(mesh.trias))
							}

							bufv := ""
							bufvt := ""
							bufvn := ""
							buff := ""

							swp := true
							skipped := 0

							for i := range mesh.trias {
								t := &mesh.trias[i]

								bufv += fmt.Sprintf("v %f %f %f\n", t.x, t.y, t.z)
								if uv {
									tx := &mesh.uvs[i]
									bufvt += fmt.Sprintf("vt %f %f\n", tx.u, 1.0-tx.v)
								}
								if vn {
									n := &mesh.norms[i]
									bufvn += fmt.Sprintf("vn %f %f %f\n", n.x, n.y, n.z)
								}

								if !t.skip {
									i2 := 1
									i3 := 2
									if swp {
										i2, i3 = i3, i2
									}

									if uv && vn {
										buff += fmt.Sprintf("f %d/%d/%d %d/%d/%d %d/%d/%d\n",
											vertIndex-i3, textIndex-i3, normIndex-i3,
											vertIndex-i2, textIndex-i2, normIndex-i3,
											vertIndex, textIndex, normIndex)
									} else if uv {
										buff += fmt.Sprintf("f %d/%d %d/%d %d/%d\n",
											vertIndex-i3, textIndex-i3, vertIndex-i2, textIndex-i2, vertIndex, textIndex)
									} else if vn {
										buff += fmt.Sprintf("f %d//%d %d//%d %d//%d\n",
											vertIndex-i3, textIndex-i3, vertIndex-i2, normIndex-i2, normIndex, normIndex)
									} else {
										buff += fmt.Sprintf("f %d %d %d\n", vertIndex-i3, vertIndex-i2, vertIndex)
									}

									swp = !swp
									if skipped == 1 {
										swp = !swp
									}
									skipped = 0
								}

								vertIndex++
								if uv {
									textIndex++
								}
								if vn {
									normIndex++
								}
							}
							if buff != "" {
								fmt.Fprintf(ofile, "o obj_%.6x\n", packet.fileStruct)
								ofile.WriteString(bufv)
								ofile.WriteString(bufvt)
								ofile.WriteString(bufvn)
								fmt.Fprintf(ofile, "usemtl mat_%d\n", object.MaterialId)
								ofile.WriteString(buff)
							}
						}
					}
				}
			}
		}
	}

	return []string{ofileName}, nil
}

func (*Mesh) ExtractFromNode(nd *wad.WadNode, outfname string) error {
	log.Printf("\n\nMesh '%s' extraction", nd.Name)

	pathPrefix := "../"
	for i := 0; i < nd.Depth; i++ {
		pathPrefix += "../"
	}

	// get path to textures files (already exported)
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

	resNames, err := mesh.ExtractObj(textures, outfname)
	if err != nil {
		return err
	}

	nd.ExtractedNames = resNames
	nd.Cache = mesh
	return nil
}
