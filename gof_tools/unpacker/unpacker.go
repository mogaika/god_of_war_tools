package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"os"
	"path"
)

type File struct {
	Name     string
	Disk     uint32
	Size     uint32
	StartSec uint32
}

func byteToString(bs []byte) string {
	n := bytes.IndexByte(bs, 0)
	if n < 0 {
		n = len(bs)
	}
	return string(bs[0:n])
}

func ParseTok(tok_file string) ([]*File, error) {
	file, err := os.OpenFile(tok_file, os.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, 24)
	files := make([]*File, 0)

	for {
		n, err := file.Read(buffer)
		if err != nil || n != 24 {
			if err == io.EOF {
				return files, nil
			} else {
				return nil, err
			}
		}

		file := &File{
			Name:     byteToString(buffer[0:12]),
			Disk:     binary.LittleEndian.Uint32(buffer[12:16]),
			Size:     binary.LittleEndian.Uint32(buffer[16:20]),
			StartSec: binary.LittleEndian.Uint32(buffer[20:24])}

		files = append(files, file)
		log.Printf("Finded file %#v\n", file)
	}
}

func Unpack(game_folder string, out_folder string) error {
	files, err := ParseTok(path.Join(game_folder, "GODOFWAR.TOC"))
	log.Printf("loaded %v files\n", len(files))
	return err
}

func main() {
	flag.Parse()
	args := flag.Args()

	game := "."
	if len(args) > 0 {
		game = args[0]
	}

	out := path.Join(game, "pack")
	if len(args) > 1 {
		out = args[1]
	}

	if err := Unpack(game, out); err != nil {
		log.Fatalln("Error when unpaking: ", err)
	}
}
