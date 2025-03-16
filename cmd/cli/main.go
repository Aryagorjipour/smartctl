package main

import (
	"fmt"
	"github.com/Aryagorjipour/smartctl/internal/tui"
	"os"
)

func main() {
	p := tui.NewProgram()
	if err := p.Start(); err != nil {
		fmt.Printf("خطا: %v\n", err)
		os.Exit(1)
	}
}
