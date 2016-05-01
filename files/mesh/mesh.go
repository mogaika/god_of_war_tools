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

const MESH_MAGIC = 0x0001000f

func init() {
	wad.PregisterExporter(MESH_MAGIC, &Mesh{})
}

func NewFromData(rdat io.Reader, debug_file_name string) (*Mesh, error) {
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

	debugOut, err := os.Create("./logs/mesh_log-" + debug_file_name + ".txt")
	if err != nil {
		return nil, err
	}
	defer debugOut.Close()
	debugLogger := log.New(debugOut, "", 0)

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

						err, packet.Blocks = VifRead1(file[packet.fileStruct:packetEnd], debugLogger, packet.fileStruct)
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

					swp := false
					for _, mesh := range packet.Blocks {
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

	mesh, err := NewFromData(reader, nd.Name)
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
