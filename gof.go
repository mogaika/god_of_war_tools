package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mogaika/god_of_war_tools/commands"
)

type Command interface {
	DefineFlags(*flag.FlagSet)
	Run() error
}

var cmds map[string]Command = map[string]Command{
	"unpack": &commands.Unpack{},
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

		err := sc.Run()
		if err != nil {
			fmt.Printf("Program exit with error: %v\n", err)
			os.Exit(2)
		} else {
			fmt.Println("Program OK")
		}
	} else {
		fmt.Printf("%s is not a valid command", cmdname)
		flag.Usage()
		os.Exit(1)
	}
}
