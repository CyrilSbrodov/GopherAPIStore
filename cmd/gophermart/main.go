package main

import (
	"github.com/CyrilSbrodov/GopherAPIStore/cmd"
)

func main() {
	srv := cmd.NewApp()
	srv.Start()
}
