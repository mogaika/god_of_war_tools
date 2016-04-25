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
	r, g, b, a float32
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
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN04D.WAD.ex\polySurface16564_0`)
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\SEWR01.WAD.ex\Sewer1b_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_BRSRK2.WAD.ex\berserkBlade_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ARENA10.WAD.ex\Arena_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_HERO0.WAD.ex\hero_0`)
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN02A.WAD.ex\scaffoldTopFac_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN01A.WAD.ex\nightSky_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_SHELL.WAD.ex\MAI_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_PERM.WAD.ex\chest_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN01B.WAD.ex\insideShip06_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN01B.WAD.ex\insideShip07_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_SHELL.WAD.ex\Visuals_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_SHELL.WAD.ex\firePlane_0`)

	// problems with shit:
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_LGHTN0.WAD.ex\lightningRadius_0`)
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_PERM.WAD.ex\HUD_0`)
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

	for i := 0; !exit; i++ {
		pos = ((pos + 3) / 4) * 4
		if pos >= uint32(len(vif)) {
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

			if pk_cmd == 0x6d && components == 4 && width == 16 && signed != 0 {
				// it is xyzc data
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
					skip := (u8(bp+7) >> 4) != 0

					bp += 8
					block.trias = append(block.trias, stXYZ{x: x, y: y, z: z, skip: skip})
				}
			} else {
				log.Printf("%s %.6x vif unpack: %.2x elements: %.2x components: %d type: %.2d target: %.3x sign: %d addr: %d size: %.4x",
					spaces, debug_off+pos, pk_cmd, pk_num, components, width, target, signed, address, blocksize)
			}

			pos += blocksize
		} else {
			switch pk_cmd {
			case 0:
				log.Printf("%s %.6x nop", spaces, debug_off+pos)
			case 01:
				log.Printf("%s %.6x Stcycl wl=%.2x cl=%.2x", spaces, debug_off+pos, pk_dat2, pk_dat1)
			case 05:
				cmode := " pos "
				/*
							enum // Decompression modes
					case 0:
							{
								Normal = 0,
								OffsetDecompression, // would conflict with vif code
								Difference
							}
				*/
				switch pk_dat1 {
				case 1:
					cmode = "[pos]"
				case 2:
					cmode = "[cur]"
				}
				log.Printf("%s %.6x Stmod  mode=%s (%d)", spaces, debug_off+pos, cmode, pk_dat1)
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
	return nil, block
}

func Convert1(mdl string) error {
	mdl = utils.PathPrepare(mdl)
	log.Printf("File `%s`", mdl)
	file, err := ioutil.ReadFile(mdl)
	if err != nil {
		return err
	}

	_, mdl_filename := path.Split(mdl)

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

	mdls := u32(8)
	log.Printf("total mdls: %d", mdls)

	//	commentStart := u32(4)

	ofilename := fmt.Sprintf("res/out_%s.obj", mdl_filename)
	ofile, err := os.Create(ofilename)
	if err != nil {
		log.Fatalf("Cannot create file %s: %v", ofilename, err)
	}
	defer ofile.Close()

	vertnum := 1

	for i_mdl := uint32(0); i_mdl < mdls; i_mdl++ {
		mbs := u32(0x50 + i_mdl*4)

		datacount := uint32(u16(mbs + 2))
		log.Printf(" current mdl: %d mbs: %x; datas: %d", i_mdl, mbs, datacount)

		fmt.Fprintf(ofile, "g group_%.6x\n", mbs)
		// data = group ?
		// sector = object ?
		// item = part of stream ?

		for i_data := uint32(0); i_data < datacount; i_data++ {
			data := mbs + u32(mbs+i_data*4+4)
			sectors := u32(data + 4)

			log.Printf("  data %d: %x; sectors: %d", i_data, data, sectors)

			for i_sec := uint32(0); i_sec < sectors; i_sec++ {
				sector := data + u32(0xc+data+i_sec*4)
				t := u16(sector)

				fmt.Fprintf(ofile, "o sec_%.6x\n", sector)
				log.Printf("   sector %d: %x; type: %.2x", i_sec, sector, t)

				// most of sectors is 0xE
				if t == 0xe || t == 0x1d || t == 0x24 {
					//t0 := u32(sector + 4)
					//t2 := sector + 0x20

					packetsCount := u32(sector+0xc) * uint32(u8(sector+0x18))

					for i := uint32(0); i < packetsCount; i++ {
						packet := sector + 0x20 + i*0x10
						rep := u32(packet + 4)
						newrep := rep + sector

						block := newrep
						swp := false
						log.Printf("    packet %d; pos: %x block:%x %s", i, packet, newrep, mdl_filename)

						err, vifpack := VifRead1(file[block:], block)
						if err != nil {
							return err
						} else {
							if vifpack.trias != nil && len(vifpack.trias) > 0 {
								for i := range vifpack.trias {
									t := &vifpack.trias[i]

									fmt.Fprintf(ofile, "v %f %f %f\n\n", t.x, t.y, t.z)

									if !t.skip {
										i2 := vertnum - 1
										i3 := vertnum - 2
										if swp {
											i2, i3 = i3, i2
										}

										fmt.Fprintf(ofile, "f %d %d %d\n", vertnum, i2, i3)
									}
									swp = !swp
									vertnum++
								}
								fmt.Fprintf(ofile, "\n\n")
							}
						}
					}
				}
			}
		}
	}
	return nil
}
