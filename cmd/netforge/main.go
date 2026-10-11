package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nixiz0/netforge/internal/dial"
	"github.com/nixiz0/netforge/internal/dns"
	"github.com/nixiz0/netforge/internal/echo"
	"github.com/nixiz0/netforge/internal/proxy"
	"github.com/nixiz0/netforge/internal/socks"
	"github.com/nixiz0/netforge/internal/tun"
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
		err = dial.Run(addrArg(defaultAddr))
	case "proxy":
		err = proxy.Run()
	case "socks":
		err = socks.Run()
	case "udpecho":
		err = echo.RunUDP(defaultAddr)
	case "udpdial":
		err = dial.RunUDP(addrArg(defaultAddr))
	case "tunnel":
		switch {
		case len(os.Args) > 2 && os.Args[2] == "server":
			err = tunnel.RunServer(defaultAddr)
		case len(os.Args) > 2 && os.Args[2] == "client":
			err = tunnel.RunClient(tunnel.ServerAddr)
		default:
			usage()
		}
	case "dns":
		err = dns.Run(addrArg(dns.Addr))
	case "tun":
		err = tun.Run()
	case "vpn":
		switch {
		case len(os.Args) > 2 && os.Args[2] == "server":
			err = vpn(":"+tunnel.VPNPort, "")
		case len(os.Args) > 3 && os.Args[2] == "client":
			err = vpn(":0", os.Args[3])
		default:
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
	fmt.Fprintln(os.Stderr, "  tunnel client   tunnel entrance on "+tunnel.ClientAddr+"/udp: wrap app datagrams, send them to "+tunnel.ServerAddr)
	fmt.Fprintln(os.Stderr, "  dns [addr]      DNS server on addr (default "+dns.Addr+"/udp), answers A for host.lab and vpn.lab")
	fmt.Fprintln(os.Stderr, "  vpn server              (Linux, sudo) VPN server: TUN "+tun.Name+" <-> UDP :"+tunnel.VPNPort+", learns its client")
	fmt.Fprintln(os.Stderr, "  vpn client <host:port>  (Linux, sudo) VPN client: TUN "+tun.Name+" <-> the server, with keepalives")
	fmt.Fprintln(os.Stderr, "  tun             (Linux, sudo) create TUN interface "+tun.Name+" and log every IP packet routed into it")
	os.Exit(2)
}

// addrArg returns the optional address argument (os.Args[2]), or def when it is missing.
func addrArg(def string) string {
	if len(os.Args) > 2 {
		return os.Args[2]
	}
	return def
}

// vpn opens the TUN interface and runs one end of the VPN: server mode when server is empty.
// main wires the two packages together, so tunnel never imports tun.
func vpn(listen, server string) error {
	dev, err := tun.Open(tun.Name)
	if err != nil {
		return err
	}
	defer dev.Close()
	return tunnel.RunTUN(dev, listen, server)
}
