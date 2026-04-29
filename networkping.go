package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const (
	icmpEchoRequest = 8
	icmpEchoReply   = 0
)

type PingResult struct {
	Success bool
	Output  string
}

type icmpPacket struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16
	Seq      uint16
	Data     []byte
}

func checksum(data []byte) uint16 {
	var sum uint32

	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i:]))
	}

	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	for (sum >> 16) > 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}

func (p *icmpPacket) Marshal() []byte {
	buf := make([]byte, 8+len(p.Data))

	buf[0] = p.Type
	buf[1] = p.Code
	binary.BigEndian.PutUint16(buf[4:], p.ID)
	binary.BigEndian.PutUint16(buf[6:], p.Seq)
	copy(buf[8:], p.Data)

	cs := checksum(buf)
	binary.BigEndian.PutUint16(buf[2:], cs)

	return buf
}

func parseICMPReply(data []byte) (*icmpPacket, error) {
	if len(data) < 28 {
		return nil, fmt.Errorf("packet too short")
	}

	icmp := data[20:]

	return &icmpPacket{
		Type:     icmp[0],
		Code:     icmp[1],
		Checksum: binary.BigEndian.Uint16(icmp[2:4]),
		ID:       binary.BigEndian.Uint16(icmp[4:6]),
		Seq:      binary.BigEndian.Uint16(icmp[6:8]),
		Data:     icmp[8:],
	}, nil
}

// Ping sends ICMP echo requests and returns output + success state.
func Ping(host string, count int, timeout time.Duration) PingResult {
	var output strings.Builder
	var success bool

	if count <= 0 {
		return PingResult{
			Success: false,
			Output:  "ping: count must be greater than zero",
		}
	}

	ipAddr, err := net.ResolveIPAddr("ip4", host)
	if err != nil {
		return PingResult{
			Success: false,
			Output:  fmt.Sprintf("ping: unknown host %s", host),
		}
	}

	conn, err := net.Dial("ip4:icmp", ipAddr.String())
	if err != nil {
		return PingResult{
			Success: false,
			Output:  fmt.Sprintf("ping: socket error: %v", err),
		}
	}
	defer conn.Close()

	fmt.Fprintf(&output, "PING %s (%s): 56 data bytes\n", host, ipAddr.String())

	var sent, received int
	var totalRTT time.Duration
	var minRTT, maxRTT time.Duration

	for i := 1; i <= count; i++ {
		sent++

		packet := &icmpPacket{
			Type: icmpEchoRequest,
			Code: 0,
			ID:   uint16(os.Getpid() & 0xffff),
			Seq:  uint16(i),
			Data: make([]byte, 56),
		}

		raw := packet.Marshal()
		start := time.Now()

		if _, err := conn.Write(raw); err != nil {
			fmt.Fprintf(&output, "Request timeout for icmp_seq %d\n", i)
			continue
		}

		_ = conn.SetReadDeadline(time.Now().Add(timeout))

		reply := make([]byte, 1500)
		n, err := conn.Read(reply)
		if err != nil {
			fmt.Fprintf(&output, "Request timeout for icmp_seq %d\n", i)
			continue
		}

		rtt := time.Since(start)

		resp, err := parseICMPReply(reply[:n])
		if err != nil || resp.Type != icmpEchoReply {
			fmt.Fprintf(&output, "Invalid reply for icmp_seq %d\n", i)
			continue
		}

		received++
		success = true
		totalRTT += rtt

		if minRTT == 0 || rtt < minRTT {
			minRTT = rtt
		}
		if rtt > maxRTT {
			maxRTT = rtt
		}

		fmt.Fprintf(&output,
			"%d bytes from %s: icmp_seq=%d ttl=64 time=%.2f ms\n",
			n-20,
			ipAddr.String(),
			resp.Seq,
			float64(rtt.Microseconds())/1000.0,
		)

		time.Sleep(1 * time.Second)
	}

	packetLoss := float64(sent-received) / float64(sent) * 100
	avgRTT := time.Duration(0)
	if received > 0 {
		avgRTT = totalRTT / time.Duration(received)
	}

	fmt.Fprintf(&output, "\n--- %s ping statistics ---\n", host)
	fmt.Fprintf(&output,
		"%d packets transmitted, %d packets received, %.1f%% packet loss\n",
		sent, received, packetLoss,
	)

	if received > 0 {
		fmt.Fprintf(&output,
			"round-trip min/avg/max = %.2f/%.2f/%.2f ms\n",
			float64(minRTT.Microseconds())/1000.0,
			float64(avgRTT.Microseconds())/1000.0,
			float64(maxRTT.Microseconds())/1000.0,
		)
	}

	return PingResult{
		Success: success,
		Output:  output.String(),
	}
}
