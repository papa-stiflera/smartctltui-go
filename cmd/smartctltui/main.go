package main

import (
	"flag"
	"os"

	"github.com/papa-stiflera/smartctltui-go/internal/smart"
	"github.com/papa-stiflera/smartctltui-go/internal/ui"
)

func main() {
	sudo := flag.Bool("sudo", false, "run with sudo (auto-restart if not root)")
	list := flag.Bool("list", false, "show device list and exit")
	mock := flag.Bool("mock", false, "demo mode with test data (no root required)")
	flag.Parse()

	if *list {
		devs, err := smart.DevicesFromScan()
		if err != nil {
			os.Exit(1)
		}
		for _, d := range devs {
			os.Stdout.WriteString(d + "\n")
		}
		return
	}

	if *mock {
		ui.Run(true)
		return
	}

	if *sudo || os.Geteuid() != 0 {
		smart.EnsureSudo()
	}

	ui.Run(false)
}
