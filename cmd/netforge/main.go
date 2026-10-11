package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nixiz0/netforge/internal/dial"
	"github.com/nixiz0/netforge/internal/echo"
	"github.com/nixiz0/netforge/internal/proxy"
	"github.com/nixiz0/netforge/internal/socks"
	"github.com/nixiz0/netforge/internal/tunnel"
)

// Theory: _networking-lessons/1-network-fundamentals.md#5-loopback
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
		err = dial.Run(addrArg())
	case "proxy":
		err = proxy.Run()
	case "socks":
		err = socks.Run()
	case "udpecho":
		err = echo.RunUDP(defaultAddr)
	case "udpdial":
		err = dial.RunUDP(addrArg())
	case "tunnel":
		if len(os.Args) > 2 && os.Args[2] == "server" {
			err = tunnel.RunServer(defaultAddr)
		} else {
			usage()
		}
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
	fmt.Fprintln(os.Stderr, "  socks           SOCKS5 server on "+socks.Addr+" (no auth, CONNECT)")
	fmt.Fprintln(os.Stderr, "  udpecho         UDP echo server on "+defaultAddr+" (same port number as echo, separate UDP port space)")
	fmt.Fprintln(os.Stderr, "  udpdial [addr]  send each typed line as one UDP datagram to addr (default "+defaultAddr+"), print replies")
	fmt.Fprintln(os.Stderr, "  tunnel server   tunnel endpoint on "+tunnel.ServerAddr+"/udp: decapsulate, forward to "+defaultAddr+"/udp, wrap replies")
	os.Exit(2)
}

// addrArg returns the optional address argument (os.Args[2]), or defaultAddr when it is missing.
func addrArg() string {
	if len(os.Args) > 2 {
		return os.Args[2]
	}
	return defaultAddr
}
