package resolve

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"
)

// Lookup describes one DNS lookup, so callers can log it.
type Lookup struct {
	Host string        // the name that was looked up
	IPs  []string      // addresses found, IPv4 first
	Took time.Duration // time spent waiting for the resolver
}

// String formats the lookup for logs: "example.com → 93.184.215.14, 2606:2800::1 in 12ms".
// Any type with a String() method is printed this way by fmt and log (the fmt.Stringer interface).
func (l *Lookup) String() string {
	return l.Host + " → " + strings.Join(l.IPs, ", ") + " in " + l.Took.Round(time.Millisecond).String()
}

// Dial connects to target ("host:port") like net.DialTimeout, but resolves a host name
// explicitly first so the lookup becomes visible. It then tries each address in turn,
// IPv4 first, until one connects, as net.Dial does internally.
//
// The returned *Lookup is nil when target already holds an IP address (no DNS query is sent).
// It is set even if every connection attempt fails afterwards.
//
// Theory: _networking-lessons/1-network-fundamentals.md#18-dns
func Dial(target string, timeout time.Duration) (net.Conn, *Lookup, error) {
	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return nil, nil, err
	}
	if net.ParseIP(host) != nil {
		conn, err := net.DialTimeout("tcp", target, timeout)
		return conn, nil, err
	}

	// Name → addresses. The OS resolver may answer from its cache, which makes repeat lookups fast.
	// Theory: _networking-lessons/1-network-fundamentals.md#dns-cache-and-ttl
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	start := time.Now()
	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, nil, err
	}
	lk := &Lookup{Host: host, IPs: ipv4First(ips), Took: time.Since(start)}

	// Try each address until one answers the handshake.
	var errs []error
	for _, ip := range lk.IPs {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), timeout)
		if err == nil {
			return conn, lk, nil
		}
		errs = append(errs, err)
	}
	return nil, lk, errors.Join(errs...)
}

// ipv4First keeps the resolver's order but moves IPv4 addresses ahead of IPv6 ones,
// because many home networks have no working IPv6 route.
func ipv4First(ips []string) []string {
	var v4, v6 []string
	for _, ip := range ips {
		if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() != nil {
			v4 = append(v4, ip)
		} else {
			v6 = append(v6, ip)
		}
	}
	return append(v4, v6...)
}
