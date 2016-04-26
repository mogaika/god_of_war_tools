package wad

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"strings"

	"github.com/mogaika/god_of_war_tools/utils"
)

const (
	NODE_TYPE_DATA = iota
	NODE_TYPE_LINK
)

type WadNode struct {
	Name     string // can be empty
	Path     string
	Parent   *WadNode
	Wad      *Wad
	Type     int // NODE_TYPE_*
	SubNodes []*WadNode
	Depth    int

	// NODE_TYPE_DATA
	Size      uint32
	Format    uint32 // first 4 bytes of data
	DataStart uint32

	// Export caches
	Extracted      bool
	Cache          interface{}
	ExtractedNames []string

	// NODE_TYPE_LINK
	LinkTo *WadNode
}

type Wad struct {
	Nodes  []*WadNode
	reader io.ReaderAt

	Version int // utils.GAME_VERSION_*
}

type WadFormatExporter interface {
	ExtractFromNode(wadnode *WadNode, outfname string) error
}

var wadExporter map[uint32]WadFormatExporter = make(map[uint32]WadFormatExporter, 0)

func PregisterExporter(format_magic uint32, exporter WadFormatExporter) {
	wadExporter[format_magic] = exporter
}

func (nd *WadNode) StringPrefixed(prefix string) string {
	switch nd.Type {
	case NODE_TYPE_DATA:
		res := fmt.Sprintf("%sdata size: 0x%.6x format: 0x%.8x start: 0x%.8x '%s'",
			prefix, nd.Size, nd.DataStart, nd.Format, nd.Name)

		if len(nd.SubNodes) > 0 {
			postfix := prefix + "  "
			res += " {\n"

			for _, n := range nd.SubNodes {
				res += fmt.Sprintf("%s\n", n.StringPrefixed(postfix))
			}
			res = fmt.Sprintf("%s%s}", res, prefix)
		}
		return res
	case NODE_TYPE_LINK:
		if nd.LinkTo != nil {
			return fmt.Sprintf("%slink '%s' -> '%s'", prefix, nd.Name, nd.LinkTo.Path)
		} else {
			return fmt.Sprintf("%slink '%s' #UNRESOLVED_LINK#", prefix, nd.Name)
		}
	}
	return prefix + "! ! ! ! unknown node type\n"
}

func (nd *WadNode) String() string {
	return nd.StringPrefixed("")
}

func (nd *WadNode) Find(name string, uptree bool) *WadNode {
	for _, v := range nd.SubNodes {
		if v.Name == name {
			return v
		}
	}
	if uptree {
		if nd.Parent != nil {
			return nd.Parent.Find(name, uptree)
		} else {
			return nd.Wad.Find(name)
		}
	}
	return nil
}

func (nd *WadNode) DataReader() (*io.SectionReader, error) {
	if nd.Type != NODE_TYPE_DATA {
		return nil, errors.New("Node must be data for reading")
	} else {
		return io.NewSectionReader(nd.Wad.reader, int64(nd.DataStart), int64(nd.Size)), nil
	}
}

func (nd *WadNode) DataRead() ([]byte, error) {
	rdr, err := nd.DataReader()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, nd.Size)
	_, err = rdr.ReadAt(buf, 0)
	if err != nil {
		return nil, err
	} else {
		return buf, nil
	}
}

func (nd *WadNode) Extract(outdir string) error {
	if nd.Type == NODE_TYPE_DATA {
		myPath := path.Join(outdir, strings.Replace(nd.Path, ":", "-", -1))

		for _, sn := range nd.SubNodes {
			if err := sn.Extract(outdir); err != nil {
				return err
			}
		}

		//	log.Printf("extracting '%s' 0x%x : 0x%x", nd.Path, nd.Format, nd.Size)
		if !nd.Extracted {
			if ex, f := wadExporter[nd.Format]; f {
				dumpfname := myPath + ".dump"

				rdr, derr := nd.DataReader()
				if derr == nil {
					var f *os.File
					f, derr = os.Create(dumpfname)
					if derr == nil {
						defer f.Close()
						_, derr = io.Copy(f, rdr)
					}
				}
				if derr != nil {
					fmt.Errorf("Error when dumping '%s' -> '%s': %v", nd.Path, dumpfname, derr)
				}

				if err := ex.ExtractFromNode(nd, myPath); err != nil {
					return fmt.Errorf("Error when extracting '%s': %v", nd.Path, err)
				}
				nd.Extracted = true
			}
		}
	}

	return nil
}

func (wad *Wad) Find(name string) *WadNode {
	for _, v := range wad.Nodes {
		if v.Name == name {
			return v
		}
	}
	return nil
}

func (wad *Wad) Extract(outdir string) error {
	for _, nd := range wad.Nodes {
		if err := nd.Extract(outdir); err != nil {
			return err
		}
	}
	return nil
}

func (wad *Wad) newNode(parent *WadNode, name string, nodeType int) *WadNode {
	node := &WadNode{
		Parent: parent,
		Type:   nodeType,
		Wad:    wad,
		Name:   name,
	}
	node.Depth = 0
	if parent != nil {
		node.Depth = parent.Depth + 1
	}
	node.Path = name
	if node.Parent != nil {
		node.Path = path.Join(node.Parent.Path, node.Path)
	}
	return node
}

func (wad *Wad) DetectVersion() (int, error) {
	wad.Version = utils.GAME_VERSION_UNKNOWN

	var buffer [4]byte
	_, err := wad.reader.ReadAt(buffer[:], 0)
	if err != nil {
		return wad.Version, err
	}

	first_tag := binary.LittleEndian.Uint32(buffer[:])
	switch first_tag {
	case 0x378:
		wad.Version = utils.GAME_VERSION_GOW_1
	case 0x15:
		wad.Version = utils.GAME_VERSION_GOW_2
	default:
		return wad.Version, errors.New("Cannot detect version")
	}
	return wad.Version, nil
}

func NewWad(f io.ReaderAt, version int) (wad *Wad, err error) {
	wad = &Wad{
		reader:  f,
		Version: version,
	}

	if version == utils.GAME_VERSION_UNKNOWN {
		wad.Version, err = wad.DetectVersion()
		if err != nil {
			return nil, err
		}
	}

	item := make([]byte, 0x20)
	newGroupTag := false
	var currentNode *WadNode

	pos := int64(0)

	for {
		const (
			PACK_STUFF = iota
			PACK_UNKNOWN
			PACK_GROUP_START
			PACK_GROUP_END
			PACK_DATA
		)
		pack_id := PACK_STUFF

		data_pos := pos + 0x20
		n, err := f.ReadAt(item, pos)
		if err != nil {
			if err == io.EOF {
				if n != 0x20 && n != 0 {
					return nil, errors.New("File end is corrupt")
				} else {
					break
				}
			} else {
				return nil, err
			}
		}

		tag := binary.LittleEndian.Uint16(item[0:2])
		size := binary.LittleEndian.Uint32(item[4:8])
		name := utils.BytesToString(item[8:32])

		switch wad.Version {
		case utils.GAME_VERSION_GOW_2:
			switch tag {
			case 0x01: // file data packet
				pack_id = PACK_DATA
			case 0x02: // file header group start
				pack_id = PACK_GROUP_START
			case 0x03: // file header group end
				pack_id = PACK_GROUP_END
			case 0x13: // file data start
				pack_id = PACK_STUFF
			case 0x15: // file header start
				pack_id = PACK_STUFF
			case 0x16: // file header pop heap
				pack_id = PACK_STUFF
			case 0: // entity count
				size = 0
				pack_id = PACK_STUFF
			}
		case utils.GAME_VERSION_GOW_1:
			switch tag {
			case 0x1e: // file data packet
				pack_id = PACK_DATA
			case 0x28: // file data group start
				pack_id = PACK_GROUP_START
			case 0x32: // file data group end
				pack_id = PACK_GROUP_END
			case 0x378: // file header start
				pack_id = PACK_STUFF
			case 0x3e7: // file header pop heap
				pack_id = PACK_STUFF
			case 0x29a: // file data start
				pack_id = PACK_STUFF
			case 0x18: // entity count
				size = 0
				pack_id = PACK_STUFF
			}
		default:
			return nil, errors.New("Unknown verison of game")
		}

		switch pack_id {
		case PACK_GROUP_END:
			newGroupTag = false
			if currentNode == nil {
				return nil, errors.New("Trying to end of not started group")
			} else {
				currentNode = currentNode.Parent
			}
		case PACK_GROUP_START:
			newGroupTag = true
		case PACK_DATA:
			var node *WadNode
			// minimal size of data == 4, for storing data format
			if size == 0 {
				node = wad.newNode(currentNode, name, NODE_TYPE_LINK)
				// Try resolve link
				if currentNode == nil {
					return nil, errors.New("Link cannot be in root node")
				}
				for cn := currentNode; cn != nil && node.LinkTo == nil; cn = cn.Parent {
					for _, v := range cn.SubNodes {
						if v.Name == node.Name {
							node.LinkTo = v
						}
					}
				}
				if node.LinkTo == nil {
					for _, v := range wad.Nodes {
						if v.Name == node.Name {
							node.LinkTo = v
						}
					}
				}

				if node.LinkTo == nil {
					return nil, fmt.Errorf(" ### Unresolved link to '%s'", node.Name)
				}
			} else {
				node = wad.newNode(currentNode, name, NODE_TYPE_DATA)

				var bfmt [4]byte
				_, err := wad.reader.ReadAt(bfmt[:], data_pos)
				if err != nil {
					return nil, err
				}
				node.Format = binary.LittleEndian.Uint32(bfmt[0:4])
			}
			node.Size = size
			node.DataStart = uint32(data_pos)

			if currentNode == nil {
				wad.Nodes = append(wad.Nodes, node)
			} else {
				currentNode.SubNodes = append(currentNode.SubNodes, node)
			}

			if newGroupTag {
				newGroupTag = false
				currentNode = node
			}
		case PACK_STUFF:
		}

		/*
			if size == 0 {
				log.Printf("%.8x:%.4x:%.8x tag  '%s'", rpos, tag, size, name)
			} else {
				log.Printf("%.8x:%.4x:%.8x data '%s'", rpos, tag, size, name)
			}
		*/

		off := (size + 15) & (15 ^ math.MaxUint32)
		pos = int64(off) + pos + 0x20
	}

	return wad, nil
}
