package tok

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"

	"github.com/mogaika/god_of_war_tools/utils"
)

type File struct {
	Pack     uint32
	Size     uint32
	StartSec uint32
	Count    int
}

type TokFile map[string]*File

func DetectVersion(tokfile io.Reader) (int, error) {
	buffer := make([]byte, 4)
	_, err := tokfile.Read(buffer)
	if err != nil {
		return utils.GAME_VERSION_UNKNOWN, err
	}

	ver := utils.GAME_VERSION_GOW_1
	strend := false
	for _, i := range buffer {
		if i == 0 {
			strend = true
		} else if i < 20 || i > 127 {
			ver = utils.GAME_VERSION_GOW_2
		} else if strend {
			ver = utils.GAME_VERSION_GOW_2
			break
		}
	}
	return ver, nil
}

// GOF 1
func parseTok1(file io.Reader) (TokFile, error) {
	buffer := make([]byte, 24)
	files := make(TokFile, 0)

	for {
		_, err := file.Read(buffer)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		name := utils.BytesToString(buffer[0:12])
		if name == "" {
			break
		}

		file := &File{
			Pack:     binary.LittleEndian.Uint32(buffer[12:16]),
			Size:     binary.LittleEndian.Uint32(buffer[16:20]),
			StartSec: binary.LittleEndian.Uint32(buffer[20:24])}

		if _, ok := files[name]; ok {
			files[name].Count++
			if files[name].Size != file.Size {
				log.Printf("File is not copy %s\n", name)
			}
		} else {
			files[name] = file
		}
	}

	return files, nil
}

// GOF 2
func parseTok2(file io.Reader) (TokFile, error) {
	const SectorsInFile = (0x3FFFF800 / utils.SectorSize)

	buffer := make([]byte, 4)

	_, err := file.Read(buffer)
	if err != nil {
		return nil, err
	}

	fcount := binary.LittleEndian.Uint32(buffer)
	maxindex := uint32(0)

	buffer = make([]byte, 36)
	files := make(TokFile)

	for i := uint32(0); i < fcount; i++ {
		_, err := file.Read(buffer)
		if err != nil {
			return nil, err
		}

		name := utils.BytesToString(buffer[0:24])
		file := &File{
			Size:     binary.LittleEndian.Uint32(buffer[24:28]),
			StartSec: binary.LittleEndian.Uint32(buffer[32:36])}

		if _, ok := files[name]; ok {
			files[name].Count++
			if files[name].Size != file.Size {
				log.Printf("File is not copy %s\n", name)
			}
		} else {
			files[name] = file
		}

		if file.StartSec > maxindex {
			maxindex = file.StartSec
		}
	}

	buffer = make([]byte, 4)
	posmap := make([]uint32, maxindex+1)
	for i := range posmap {
		_, err := file.Read(buffer)
		if err != nil {
			return nil, err
		}
		sz := binary.LittleEndian.Uint32(buffer)
		posmap[i] = sz
	}

	for _, f := range files {
		pos := posmap[f.StartSec]
		f.StartSec = pos % SectorsInFile
		f.Pack = pos / SectorsInFile
	}

	return files, nil
}

func Decode(file io.ReadSeeker, version int) (files TokFile, err error) {
	if version == utils.GAME_VERSION_UNKNOWN {
		version, err = DetectVersion(file)
		file.Seek(0, os.SEEK_SET)
		if err != nil {
			return
		}
		log.Printf("Detected tok version: %v\n", version)
	}

	switch version {
	case utils.GAME_VERSION_GOW_1:
		return parseTok1(file)
	case utils.GAME_VERSION_GOW_2:
		return parseTok2(file)
	case utils.GAME_VERSION_UNKNOWN:
		return nil, errors.New("Unknown tok version for parsing")
	}
	return
}
