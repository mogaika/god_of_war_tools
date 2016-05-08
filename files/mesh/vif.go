package mesh

import (
	"encoding/binary"
	"fmt"
	"io"
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

// GS use 12:4 fixed point format
// 1 << 4 = 16
const GSFixed12Point4Delimeter = 16.0
const GSFixed12Point4Delimeter1000 = 4096.0

func VifRead1(vif []byte, debug_off uint32, debugOut io.Writer) (error, []*stBlock) {
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
							bp := pos + uint32(i)*0x10
							fmt.Fprintf(debugOut, "%s -  %.6x = %.4x %.4x %.4x %.4x  %.4x %.4x %.4x %.4x\n", spaces, debug_off+bp,
								binary.LittleEndian.Uint16(vif[bp:bp+2]), binary.LittleEndian.Uint16(vif[bp+2:bp+4]),
								binary.LittleEndian.Uint16(vif[bp+4:bp+6]), binary.LittleEndian.Uint16(vif[bp+6:bp+8]),
								binary.LittleEndian.Uint16(vif[bp+8:bp+10]), binary.LittleEndian.Uint16(vif[bp+10:bp+12]),
								binary.LittleEndian.Uint16(vif[bp+12:bp+14]), binary.LittleEndian.Uint16(vif[bp+14:bp+16]))
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
				fmt.Fprintf(debugOut, "%s %.6x vif unpack [%s]: %.2x elements: %.2x components: %d type: %.2d target: %.3x sign: %t addr: %t size: %.6x\n",
					spaces, debug_off+tagpos, handledBy, pk_cmd, pk_num, components, width, target, signed, address, blocksize)
			}

			pos += blocksize
		} else {
			switch pk_cmd {
			case 0:
				fmt.Fprintf(debugOut, "%s %.6x nop\n", spaces, debug_off+tagpos)
			case 01:
				fmt.Fprintf(debugOut, "%s %.6x Stcycl wl=%.2x cl=%.2x\n", spaces, debug_off+tagpos, pk_dat2, pk_dat1)
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
				fmt.Fprintf(debugOut, "%s %.6x Stmod  mode=%s (%d)\n", spaces, debug_off+tagpos, cmode, pk_dat1)
			case 0x14:
				fmt.Fprintf(debugOut, "%s %.6x Mscall proc command\n", spaces, debug_off+tagpos)
				flush = true
			case 0x30:
				fmt.Fprintf(debugOut, "%s %.6x Strow  proc command\n", spaces, debug_off+tagpos)
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
							u.u = float32(int16(binary.LittleEndian.Uint16(block_data_uv[bp:bp+2]))) / GSFixed12Point4Delimeter1000
							u.v = float32(int16(binary.LittleEndian.Uint16(block_data_uv[bp+2:bp+4]))) / GSFixed12Point4Delimeter1000
						}
					case 4:
						currentBlock.uvs = make([]stUV, len(block_data_uv)/8)
						for i := range currentBlock.trias {
							bp := i * 8
							u := &currentBlock.uvs[i]
							u.u = float32(int32(binary.LittleEndian.Uint32(block_data_uv[bp:bp+4]))) / GSFixed12Point4Delimeter1000
							u.v = float32(int32(binary.LittleEndian.Uint32(block_data_uv[bp+4:bp+8]))) / GSFixed12Point4Delimeter1000
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

				fmt.Fprintf(debugOut, "%s = Flush xyzw:%t, rgba:%t, uv:%t, norm:%t\n", spaces,
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
