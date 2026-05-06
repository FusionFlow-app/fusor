package main

import (
	"log"

	"fusor/internal/tui"
)

func main() {
	p := tui.NewProgram()

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
