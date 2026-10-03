package main

import (
	"fmt"
	"os"
)

const defaultAddr = "127.0.0.1:9000" // loopback

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	switch os.Args[1] {
	case "echo":
		runEchoServer(defaultAddr)
	case "dial":
		addr := defaultAddr
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		runDial(addr)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: netforge <command>")
	fmt.Fprintln(os.Stderr, "  echo          TCP echo server on "+defaultAddr)
	fmt.Fprintln(os.Stderr, "  dial [addr]   connect to addr (default "+defaultAddr+"), type to send, replies are printed")
	os.Exit(2)
}
