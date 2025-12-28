package ping

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const echoData = "deadman-go"

// ICMPPinger sends ICMP echo requests using raw sockets.
type ICMPPinger struct {
	id  int
	seq uint32
}

// NewICMPPinger initializes a pinger with a process-scoped identifier.
func NewICMPPinger() (*ICMPPinger, error) {
	return &ICMPPinger{id: os.Getpid() & 0xffff}, nil
}

// Ping sends one ICMP echo request and waits for the reply.
func (p *ICMPPinger) Ping(ctx context.Context, addr string, timeout time.Duration) Result {
	if err := ctx.Err(); err != nil {
		return Result{Success: false, Error: err}
	}

	ip, ipNet, err := resolveIP(addr)
	if err != nil {
		return Result{Success: false, Error: err}
	}

	network, protocol, requestType, replyType := icmpSettings(ipNet)
	conn, err := icmp.ListenPacket(network, "")
	if err != nil {
		return Result{Success: false, Error: err}
	}
	defer conn.Close()

	seq := int(atomic.AddUint32(&p.seq, 1))
	msg := icmp.Message{
		Type: requestType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   p.id,
			Seq:  seq,
			Data: []byte(echoData),
		},
	}

	payload, err := msg.Marshal(nil)
	if err != nil {
		return Result{Success: false, Error: err}
	}

	deadline := effectiveDeadline(ctx, timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		return Result{Success: false, Error: err}
	}

	start := time.Now()
	if _, err := conn.WriteTo(payload, ip); err != nil {
		return Result{Success: false, Error: err}
	}

	buf := make([]byte, 1500)
	for {
		if err := ctx.Err(); err != nil {
			return Result{Success: false, Error: err}
		}

		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			// timeoutエラーを適切に処理
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return Result{Success: false, Error: fmt.Errorf("ping timeout: %w", err)}
			}
			return Result{Success: false, Error: err}
		}
		if peer == nil {
			continue
		}

		reply, err := icmp.ParseMessage(protocol, buf[:n])
		if err != nil {
			continue
		}
		if reply.Type != replyType {
			continue
		}
		body, ok := reply.Body.(*icmp.Echo)
		if !ok {
			continue
		}
		if body.ID != p.id || body.Seq != seq {
			continue
		}

		return Result{Success: true, RTT: time.Since(start)}
	}
}

func resolveIP(addr string) (*net.IPAddr, net.IP, error) {
	ipAddr, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		return nil, nil, err
	}
	if ipAddr.IP == nil {
		return nil, nil, fmt.Errorf("invalid IP address: %s", addr)
	}
	return ipAddr, ipAddr.IP, nil
}

func icmpSettings(ip net.IP) (network string, protocol int, requestType icmp.Type, replyType icmp.Type) {
	if ip.To4() != nil {
		return "ip4:icmp", ipv4.ICMPTypeEcho.Protocol(), ipv4.ICMPTypeEcho, ipv4.ICMPTypeEchoReply
	}
	return "ip6:ipv6-icmp", ipv6.ICMPTypeEchoRequest.Protocol(), ipv6.ICMPTypeEchoRequest, ipv6.ICMPTypeEchoReply
}

func effectiveDeadline(ctx context.Context, timeout time.Duration) time.Time {
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		return ctxDeadline
	}
	return deadline
}
