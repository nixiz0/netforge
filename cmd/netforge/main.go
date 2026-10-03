package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nixiz0/netforge-go/internal/dial"
	"github.com/nixiz0/netforge-go/internal/echo"
	"github.com/nixiz0/netforge-go/internal/proxy"
)

const defaultAddr = "127.0.0.1:9000" // loopback

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	var err error
	switch os.Args[1] {
	case "echo":
		err = echo.Run(defaultAddr)
	case "dial":
		addr := defaultAddr
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		err = dial.Run(addr)
	case "proxy":
		err = proxy.Run()
	default:
		usage()
	}

	// Packages never exit the program themselves: they return the error, main decides.
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: netforge <command>")
	fmt.Fprintln(os.Stderr, "  echo            TCP echo server on "+defaultAddr)
	fmt.Fprintln(os.Stderr, "  dial [addr]     connect to addr (default "+defaultAddr+"), type to send, replies are printed")
	fmt.Fprintln(os.Stderr, "  proxy           HTTP forward proxy on "+proxy.Addr)
	os.Exit(2)
}
