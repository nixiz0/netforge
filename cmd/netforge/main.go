package main

import (
	"fmt"
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

	switch os.Args[1] {
	case "echo":
		echo.Run(defaultAddr)
	case "dial":
		addr := defaultAddr
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		dial.Run(addr)
	case "proxy":
		proxy.Run()
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: netforge <command>")
	fmt.Fprintln(os.Stderr, "  echo            TCP echo server on "+defaultAddr)
	fmt.Fprintln(os.Stderr, "  dial [addr]     connect to addr (default "+defaultAddr+"), type to send, replies are printed")
	fmt.Fprintln(os.Stderr, "  proxy           HTTP forward proxy on "+proxy.Addr)
	os.Exit(2)
}
