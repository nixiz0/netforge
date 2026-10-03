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
		runDial(defaultAddr)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: netforge <command>")
	fmt.Fprintln(os.Stderr, "  echo   TCP echo server on "+defaultAddr)
	fmt.Fprintln(os.Stderr, "  dial   connect to "+defaultAddr+", send one message, print the reply")
	os.Exit(2)
}
