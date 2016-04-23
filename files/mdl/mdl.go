package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
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

func main() {
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ATHN05AA.WAD.ex\GenericFightloop1_0`, "genfl1.obj")
	Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\ARENA10.WAD.ex\Arena_0`, "arena.obj")
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_HERO0.WAD.ex\hero_0`, "hero.obj")
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_SHELL.WAD.ex\MAI_0`, "mai.obj")
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_PERM.WAD.ex\HUD_0`, "mai.obj")
	//Convert1(`E:\Downloads\God of War  NTSC(USA)  PS2DVD-9\unpacked\R_PERM.WAD.ex\chest_0`, "mai.obj")
}

func Convert1(mdl string, out string) error {
	log.Printf("File `%s`", mdl)
	file, err := ioutil.ReadFile(mdl)
	if err != nil {
		return err
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

	if u32(0) != 0x1000f {
		return fmt.Errorf("Unknown mdl type")
	}

	mdls := u32(8)
	//log.Printf("total mdls: %d", mdls)

	for i_mdl := uint32(0); i_mdl < mdls; i_mdl++ {
		mbs := u32(0x50 + i_mdl*4)

		datacount := uint32(u16(mbs + 2))
		//log.Printf(" current mdl: %d mbs: %x; datas: %d", i_mdl, mbs, datacount)

		for i_data := uint32(0); i_data < datacount; i_data++ {
			data := mbs + u32(mbs+i_data*4+4)
			sectors := u32(data + 4)

			//log.Printf("  data %d: %x; sectors: %d", i_data, data, sectors)

			for i_sec := uint32(0); i_sec < sectors; i_sec++ {
				sector := data + u32(0xc+data+i_sec*4)
				t := u16(sector)

				//log.Printf("   sector %d: %x; type: %.2x", i_sec, sector, t)

				// most of sectors is 0xE
				if t == 0xe || t == 0x1d || t == 0x24 {
					//t0 := u32(sector + 4)
					//t2 := sector + 0x20

					itemsCount := u32(sector+0xc) * uint32(u8(sector+0x18))

					//	log.Printf("    sector items: %x", itemsCount)

					for i := uint32(0); i < itemsCount; i++ {
						item := sector + 0x20 + i*0x10
						rep := u32(item + 4)
						newrep := rep + sector

						log.Printf("     item %d; pos: %x; replace %x to %x", i, item, rep, newrep)
						log.Printf("     item val %.8x %.8x %.8x %.8x",
							u32(newrep), u32(newrep+4), u32(newrep+8), u32(newrep+0xc))

					}
				}
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
