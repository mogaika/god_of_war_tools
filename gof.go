package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mogaika/god_of_war_tools/commands"
	_ "github.com/mogaika/god_of_war_tools/files"
)

type Command interface {
	DefineFlags(*flag.FlagSet)
	Run() error
}

var cmds map[string]Command = map[string]Command{
	"unpack":  &commands.Unpack{},
	"extract": &commands.Extract{},
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: god_of_war_tools command [arguments]")
		fmt.Println("Help: god_of_war_tools command --help")
		fmt.Println("Commands:")
		for i, _ := range cmds {
			fmt.Printf("  %s\n", i)
		}
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(0)
	}

	cmdname := flag.Arg(0)
	if sc, ok := cmds[cmdname]; ok {
		fs := flag.NewFlagSet(cmdname, flag.ExitOnError)
		sc.DefineFlags(fs)
		fs.Parse(flag.Args()[1:])

		if err := sc.Run(); err != nil {
			log.Printf("Program exit with error: %v\n", err)
			os.Exit(2)
		}
	} else {
		log.Printf("%s is not a valid command", cmdname)
		flag.Usage()
		os.Exit(1)
	}
}
