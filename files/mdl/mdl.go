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

/*
MDL
{
	+8		u32		mdls left in file
	+0x50	u32		first section offset
	SECTION
	{
	+2		u8		1 if present?, 0 if last?
	+4		u32		position realtive to first section, if 0 - last?
	}
}
*/

type stUV struct {
	u, v int16
}

type stXYZ struct {
	x, y, z float32
}

type stFACE struct {
	i, j, k int
}

func main() {
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

	commentStart := u32(4)

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
				swp := false
				log.Printf("   sector %d: %x; type: %.2x", i_sec, sector, t)

				o_tvs := 0
				o_xyzs := 0

				// most of sectors is 0xE
				if t == 0xe || t == 0x1d || t == 0x24 {
					//t0 := u32(sector + 4)
					//t2 := sector + 0x20

					itemsCount := u32(sector+0xc) * uint32(u8(sector+0x18))

					log.Printf("   sector items: %x", itemsCount)

					for i := uint32(0); i < itemsCount; i++ {
						item := sector + 0x20 + i*0x10
						rep := u32(item + 4)
						newrep := rep + sector

						block := newrep

						log.Printf("    item %d; pos: %x; replace %x to %x %s", i, item, rep, newrep, mdl_filename)

						sblock := block

					blockcycle:
						for {
							if block >= commentStart || block+3 >= uint32(len(file)) {
								log.Printf("       [      -%.6x] limit", block)
								break
							}

							dattype := u8(block + 3)
							if dattype == 0 {
								log.Printf("       [      -%.6x] zerotype", block)
								break
							}

							bcount := uint32(u8(block + 2))

							sblock = block

							switch dattype {
							case 1:
								block += 4
								//curst = sblock
								log.Printf("       [%.6x-%.6x] data %.2x      %.8x", sblock, block, dattype, u32(sblock))
							case 5:
								block += 4
								log.Printf("       [%.6x-%.6x] data %.2x      %.8x", sblock, block, dattype, u32(sblock))
							case 0x30:
								block += 4 + 16
								log.Printf("       [%.6x-%.6x] data %.2x      %.8x", sblock, block, dattype, u32(sblock))
							case 0x64:
								block += 4 + bcount*8
								log.Printf("       [%.6x-%.6x] data b;en %.2x %.8x", sblock, block, bcount, u32(sblock))
							case 0x65:
								// 2ByteSignedIntegerUVcoords(U,V)
								block += 4 + bcount*4
								log.Printf("       [%.6x-%.6x] data uv   %.2x %.8x", sblock, block, bcount, u32(sblock))

								p := sblock + 4
								for i := uint32(0); i < bcount; i++ {
									//_ := float32(int16(u16(p))) / 256.0
									//_ := float32(int16(u16(p+2))) / 256.0
									p += 4
									o_tvs++
								}

							case 0x6a:
								// 1ByteSignedInteger(X,Y,Z)
								block += 4 + ((bcount*3+3)/4)*4
								log.Printf("       [%.6x-%.6x] data wtf  %.2x %.8x", sblock, block, bcount, u32(sblock))
							case 0x6c:
								// unknown shit?
								shid := u8(sblock)
								block += 4 + bcount*0x10

								algn := ""

								switch shid {
								case 0x49:
									fallthrough
								case 0x55:
									fallthrough
								case 0x9f:
									fallthrough
								case 0xab:
									fallthrough
								case 0xf4:
									if sblock&0xf < 8 {
										block = ((block + 0x7) / 0x8) * 0x8
									}

								case 0:
									if bcount > 1 {
										block = ((block + 0xf) / 0x10) * 0x10
										algn = "bcoun"
									}
								default:
									log.Fatalf("Error: new type of shit: %.2x", shid)
								}

								log.Printf("       [%.6x-%.6x] data shit %.2x %.8X shid: %.2x off al:%s",
									sblock, block, bcount, u32(sblock), shid, algn)
							case 0x6d:
								// 2ByteSignedVertexes(X,Y,Z)+1ByteUnknown+1ByteCONN
								block += 4 + bcount*8
								log.Printf("       [%.6x-%.6x] data xyz  %.2x %.8x", sblock, block, bcount, u32(sblock))

								// GS use 12:4 fixed point format
								// 1 << 12 = 4096
								const delimetr = 4096.0
								for i := uint32(0); i < bcount; i++ {
									p := sblock + 4 + i*8
									x := float32(int16(u16(p))) / delimetr
									y := float32(int16(u16(p+2))) / delimetr
									z := float32(int16(u16(p+4))) / delimetr

									push := u8(p+7) >> 4
									fmt.Fprintf(ofile, "v %f %f %f\n#%x\n", x, y, z, push)

									if push != 8 {
										i2 := vertnum - 1
										i3 := vertnum - 2
										if swp {
											i2, i3 = i3, i2
										}

										fmt.Fprintf(ofile, "f %d %d %d\n", vertnum, i2, i3)
									}
									swp = !swp
									o_xyzs++
									vertnum++
								}

								fmt.Fprintf(ofile, "\n\n\n")
							case 0x6e:
								block += 4 + bcount*4
								log.Printf("       [%.6x-%.6x] data rgba %.2x %.8x", sblock, block, bcount, u32(sblock))
							/*case 0x3f:
							block += 4 + bcount*8
							log.Printf("       [%.6x-%.6x] data kek  %.2x %.8x", sblock, block, bcount, u32(sblock))*/
							default:
								log.Printf("  !!!!  ~~~~ UNKNOWN DATTYPE: %.6x %x[%.8x] !!!!!!!!!!!!!!!!! %s", block, dattype, u32(sblock), mdl_filename)
								break blockcycle
							}
						}
					}
				}

				log.Printf("Object vts: %d xyzs: %d\n", o_tvs, o_xyzs)
			}
		}
	}
	/*
		block := binary.LittleEndian.Uint32(buf4)
		blockid := (block >> 24) & 0xff

		size := uint32(math.MaxUint32)

		switch blockid {
		case 0x05:
		case 0x65:
		case 0x6c:
		case 0x6d:
		case 0x6e:
		case 0xd8:
		default:
			if _, err := f.Read(buf4); err != nil {
				return err
			}
			size = binary.LittleEndian.Uint32(buf4)
		}

		if block == 0x10001 {
			// size not -= 4
		} else if blockid == 0x05 {
			size = 20
		} else if blockid == 0x30 {
			size -= 0x20
			size -= 4
		} else if blockid == 0x65 {
			count := (block >> 16) & 0xff
			log.Printf("Count of uv : %v\n", count)
			size = count * 4
		} else if blockid == 0x6d {
			count := (block >> 16) & 0xff
			log.Printf("Count of xyz : %v\n", count)
			size = count*8 + 4
		} else if blockid == 0x6e {
			count := (block >> 16) & 0xff
			log.Printf("Count of colorblend? : %v\n", count)
			size = count*4 + 4
		} else if blockid == 0x6c {
			size = 16
		} else {
			size -= 4
		}

		log.Printf("block 0x%.8x [%.2x]: size 0x%.8x\n", block, blockid, size)

		if _, err := f.Seek(int64(size), os.SEEK_CUR); err != nil {
			return err
		}*/

	//log.Printf("%#v\n", head)

	return nil
}
