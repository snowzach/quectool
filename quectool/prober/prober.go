package prober

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/snowzach/quectool/quectool"
)

type Response struct {
	Success    bool              `json:"success"`
	StatusCode int               `json:"status_code,omitempty"`
	Duration   quectool.Duration `json:"duration,omitempty"`
}

// httpClient is shared across ProbeHTTP calls so we reuse connections and
// don't allocate a new Transport per probe.
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives:     false,
		MaxIdleConns:          4,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// ProbeHTTP performs a single GET to target and reports success based on a
// 2xx response within timeout.
func ProbeHTTP(ctx context.Context, target string, timeout time.Duration) (Response, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(cctx, http.MethodGet, target, nil)
	if err != nil {
		return Response{}, fmt.Errorf("invalid target: %w", err)
	}

	start := time.Now()
	resp, err := httpClient.Do(req)
	dur := time.Since(start)
	if err != nil {
		// Distinguish context errors from network errors? Caller doesn't care:
		// any failure is just Success=false.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return Response{Success: false, Duration: quectool.Duration(dur)}, nil
		}
		return Response{Success: false, Duration: quectool.Duration(dur)}, nil
	}
	// Drain a small bounded amount so the connection can be reused.
	_, _ = io.CopyN(io.Discard, resp.Body, 1<<10)
	_ = resp.Body.Close()

	return Response{
		Success:    resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode: resp.StatusCode,
		Duration:   quectool.Duration(dur),
	}, nil
}

// pingSeq is a process-global sequence number for matching echo replies.
var pingSeq uint32

// ProbePing sends a single ICMP echo to target and waits up to timeout for
// the matching reply. It tries unprivileged ICMP first ("udp4") and falls
// back to raw sockets ("ip4:icmp") if the kernel rejects the unprivileged
// form. IPv6 is supported automatically based on the resolved address.
func ProbePing(ctx context.Context, target string, timeout time.Duration) (Response, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Resolve target. We deliberately use ResolveIPAddr (not LookupHost) so
	// we get a single concrete address with no goroutine fan-out.
	resolver := &net.Resolver{}
	addrs, err := resolver.LookupIP(cctx, "ip", target)
	if err != nil || len(addrs) == 0 {
		return Response{Success: false}, nil
	}
	addr := addrs[0]
	ipv6mode := addr.To4() == nil

	conn, isPrivileged, err := dialICMP(ipv6mode)
	if err != nil {
		return Response{Success: false}, fmt.Errorf("icmp listen: %w", err)
	}
	defer conn.Close()

	// Build echo request. Use process PID (low 16 bits) as ID for raw sockets;
	// the kernel rewrites ID for unprivileged ("udp4") sockets.
	id := os.Getpid() & 0xffff
	seq := int(atomic.AddUint32(&pingSeq, 1) & 0xffff)

	var msgType icmp.Type
	if ipv6mode {
		msgType = ipv6.ICMPTypeEchoRequest
	} else {
		msgType = ipv4.ICMPTypeEcho
	}

	// 8-byte payload: send timestamp for RTT cross-check; mostly we use the
	// receive time minus 'start' below.
	payload := make([]byte, 8)
	binary.BigEndian.PutUint64(payload, uint64(time.Now().UnixNano()))

	msg := icmp.Message{
		Type: msgType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   id,
			Seq:  seq,
			Data: payload,
		},
	}
	wb, err := msg.Marshal(nil)
	if err != nil {
		return Response{Success: false}, fmt.Errorf("icmp marshal: %w", err)
	}

	dst := &net.UDPAddr{IP: addr}
	var dstAddr net.Addr = dst
	if isPrivileged {
		dstAddr = &net.IPAddr{IP: addr}
	}

	if d, ok := cctx.Deadline(); ok {
		_ = conn.SetDeadline(d)
	}

	start := time.Now()
	if _, err := conn.WriteTo(wb, dstAddr); err != nil {
		return Response{Success: false, Duration: quectool.Duration(time.Since(start))}, nil
	}

	rb := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(rb)
		if err != nil {
			return Response{Success: false, Duration: quectool.Duration(time.Since(start))}, nil
		}

		// Determine protocol number for parsing.
		var proto int
		if ipv6mode {
			proto = 58 // ICMPv6
		} else {
			proto = 1 // ICMPv4
		}
		rm, err := icmp.ParseMessage(proto, rb[:n])
		if err != nil {
			continue
		}
		// Match echo reply with our sequence number. ID is rewritten by the
		// kernel on unprivileged sockets, so don't strictly require it.
		if echo, ok := rm.Body.(*icmp.Echo); ok && echo.Seq == seq {
			isReply := (!ipv6mode && rm.Type == ipv4.ICMPTypeEchoReply) ||
				(ipv6mode && rm.Type == ipv6.ICMPTypeEchoReply)
			if isReply {
				return Response{
					Success:  true,
					Duration: quectool.Duration(time.Since(start)),
				}, nil
			}
		}
		// Otherwise keep reading until deadline.
	}
}

// dialICMP opens an ICMP packet conn, preferring unprivileged ("udp4"/"udp6")
// and falling back to raw sockets if the kernel doesn't permit the
// unprivileged form (returns EACCES or EPERM). On non-Linux it goes straight
// to raw sockets, since unprivileged ICMP is Linux-specific.
func dialICMP(ipv6mode bool) (*icmp.PacketConn, bool, error) {
	if runtime.GOOS == "linux" {
		network := "udp4"
		bind := "0.0.0.0"
		if ipv6mode {
			network = "udp6"
			bind = "::"
		}
		if c, err := icmp.ListenPacket(network, bind); err == nil {
			return c, false, nil
		}
		// Fall through to raw sockets.
	}
	network := "ip4:icmp"
	bind := "0.0.0.0"
	if ipv6mode {
		network = "ip6:ipv6-icmp"
		bind = "::"
	}
	c, err := icmp.ListenPacket(network, bind)
	if err != nil {
		return nil, false, err
	}
	return c, true, nil
}
