package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/mogaika/god_of_war_tools/utils"
)

type stUV struct {
	u, v int16
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

func main() {
	for i := 1; i < len(os.Args); i++ {
		Convert1(os.Args[i])
	}
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN04D.WAD.ex\polySurface16564_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\SEWR01.WAD.ex\Sewer1b_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_BRSRK2.WAD.ex\berserkBlade_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ARENA10.WAD.ex\Arena_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_HERO0.WAD.ex\hero_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN02A.WAD.ex\scaffoldTopFac_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN01A.WAD.ex\nightSky_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_SHELL.WAD.ex\MAI_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_PERM.WAD.ex\chest_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN01B.WAD.ex\insideShip06_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN01B.WAD.ex\insideShip07_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_SHELL.WAD.ex\Visuals_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_SHELL.WAD.ex\firePlane_0`)

	// problems with shit:
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_LGHTN0.WAD.ex\lightningRadius_0`)
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_PERM.WAD.ex\HUD_0`)
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_MEDHD2.WAD.ex\medheadNuke_0`)
}

func VifRead1(vif []byte, debug_off uint32) (error, *stBlock) {
	block := new(stBlock)
	/*u32 := func(idx uint32) uint32 {
		return binary.LittleEndian.Uint32(vif[idx : idx+4])
	}*/
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
					if block.trias == nil {
						block.trias = make([]stXYZ, 0)
					}
					// GS use 12:4 fixed point format
					// 1 << 12 = 4096
					const delimetr = 4096.0

					bp := pos
					for i := uint8(0); i < pk_num; i++ {

						x := float32(int16(u16(bp))) / delimetr
						y := float32(int16(u16(bp+2))) / delimetr
						z := float32(int16(u16(bp+4))) / delimetr
						skip := u8(bp+7)&0x80 != 0

						block.trias = append(block.trias, stXYZ{x: x, y: y, z: z, skip: skip})

						//log.Printf(" -- %.4x %+2.4f %+2.4f %+2.4f", u16(bp+6), x, y, z)

						bp += 8
					}

					grabbedType = " xyz"
				} else if pk_cmd == 0x6e && components == 4 && width == 8 && signed == 0 {
					if block.blend == nil {
						block.blend = make([]stRGBA, 0)
					}
					bp := pos
					for i := uint8(0); i < pk_num; i++ {
						block.blend = append(block.blend,
							stRGBA{r: u8(bp), g: u8(bp + 1), b: u8(bp + 2), a: u8(bp + 3)})

						bp += 4
					}

					grabbedType = "rgba"
				} else if pk_cmd == 0x65 && components == 2 && width == 16 && signed == 1 {
					if block.uvs == nil {
						block.uvs = make([]stUV, 0)
					}
					bp := pos
					for i := uint8(0); i < pk_num; i++ {
						block.uvs = append(block.uvs,
							stUV{u: int16(u16(bp)), v: int16(u16(bp + 2))})

						bp += 4
					}

					grabbedType = " uv "
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
				// log.Printf("%s %.6x Mscall proc command", spaces, debug_off+pos)
			case 0x30:
				// log.Printf("%s %.6x Strow  proc command", spaces, debug_off+pos)
			default:
				log.Printf("%s %.6x VIF command: %.2x:%.2x data: %.2x:%.2x", spaces, debug_off+pos, pk_cmd, pk_num, pk_dat1, pk_dat2)
				exit = true
			}
		}

	}
	return nil, block
}

type ModelPacket struct {
	fileStruct uint32
	Blocks     []stBlock
}

type ModelObject struct {
	fileStruct uint32
	Type       uint16
	Packets    []ModelPacket
}

type ModelGroup struct {
	fileStruct uint32
	Objects    []ModelObject
}

type ModelPart struct {
	fileStruct uint32
	Groups     []ModelGroup
}

func Convert1(mdl string) error {
	mdl = utils.PathPrepare(mdl)
	log.Printf("File `%s`", mdl)
	file, err := ioutil.ReadFile(mdl)
	if err != nil {
		return err
	}

	_, mdlFilename := path.Split(mdl)

	u32 := func(idx uint32) uint32 {
		return binary.LittleEndian.Uint32(file[idx : idx+4])
	}
	u16 := func(idx uint32) uint16 {
		return binary.LittleEndian.Uint16(file[idx : idx+2])
	}
	u8 := func(idx uint32) uint8 {
		return file[idx]
	}

	if u32(0) != 0x1000f {
		return fmt.Errorf("Unknown mdl type")
	}

	mdlCommentStart := u32(4)

	partsCount := u32(8)

	parts := make([]ModelPart, partsCount)

	log.Printf("parts: %d", partsCount)

	// build tree for blocks boundary finding
	for iPart := uint32(0); iPart < partsCount; iPart++ {
		pPart := u32(0x50 + iPart*4)
		groupsCount := uint32(u16(pPart + 2))

		parts[iPart].fileStruct = pPart
		groups := make([]ModelGroup, groupsCount)

		for iGroup := uint32(0); iGroup < groupsCount; iGroup++ {
			pGroup := pPart + u32(pPart+iGroup*4+4)
			objectsCount := u32(pGroup + 4)

			groups[iGroup].fileStruct = pGroup
			objects := make([]ModelObject, objectsCount)
			for iObject := uint32(0); iObject < objectsCount; iObject++ {
				pObject := pGroup + u32(0xc+pGroup+iObject*4)
				tObject := u16(pObject)
				packetsCount := u32(pObject+0xc) * uint32(u8(pObject+0x18))

				objects[iObject].fileStruct = pObject
				objects[iObject].Type = tObject
				packets := make([]ModelPacket, packetsCount)

				/*
					0x1d - surface mesh (bridge, skybox)
					0x0e - model mesh (ship, hero, enemy)
				*/

				if tObject == 0xe || tObject == 0x1d || tObject == 0x24 {
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

	ofileName := fmt.Sprintf("res/out_%s.obj", mdlFilename)
	ofile, err := os.Create(ofileName)
	if err != nil {
		log.Fatalf("Cannot create file %s: %v", ofileName, err)
	}
	defer ofile.Close()
	vertIndex := 1

	pointerEnd := mdlCommentStart
	for iPart := len(parts) - 1; iPart >= 0; iPart-- {
		part := &parts[iPart]
		groups := part.Groups

		log.Printf(" part: %d pos: %.6x; groups: %d", iPart, part.fileStruct, len(groups))

		for iGroup := len(groups) - 1; iGroup >= 0; iGroup-- {
			group := &groups[iGroup]
			objects := group.Objects

			log.Printf("  group: %d pos: %.6x; objects: %d", iGroup, group.fileStruct, len(objects))
			fmt.Fprintf(ofile, "g group_%.6x\n", group.fileStruct)

			for iObject := len(objects) - 1; iObject >= 0; iObject-- {
				object := &objects[iObject]
				packets := object.Packets

				log.Printf("   object: %d pos: %.6x; type: %.2x", iObject, object.fileStruct, object.Type)
				fmt.Fprintf(ofile, "o obj_%.6x\n", object.fileStruct)
				swp := true

				for iPacket := len(packets) - 1; iPacket >= 0; iPacket-- {
					packet := &packets[iPacket]

					log.Printf("    packet: %d pos: %.6x;", iPacket, packet.fileStruct)

					err, vifpack := VifRead1(file[packet.fileStruct:pointerEnd], packet.fileStruct)
					if err != nil {
						log.Printf("ERROR when vif reading: %v", err)
					} else {
						if vifpack.trias != nil && len(vifpack.trias) > 0 {

							for i := range vifpack.trias {
								t := &vifpack.trias[i]

								fmt.Fprintf(ofile, "v %f %f %f\n\n", t.x, t.y, t.z)

								if !t.skip {
									i2 := vertIndex - 1
									i3 := vertIndex - 2
									if swp {
										i2, i3 = i3, i2
									}

									fmt.Fprintf(ofile, "f %d %d %d\n", vertIndex, i2, i3)
								}
								swp = !swp
								vertIndex++
							}
							fmt.Fprintf(ofile, "\n\n")
						}
					}
					pointerEnd = packet.fileStruct
				}
				pointerEnd = object.fileStruct
			}
			pointerEnd = group.fileStruct
		}
		pointerEnd = part.fileStruct
	}
	return nil
}
