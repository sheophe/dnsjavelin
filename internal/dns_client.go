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

func (c DNSClient) SendJunkDomainsRequest(nQuestions int) (rtt time.Duration, dnsError string, err error) {
	m := new(dns.Msg)
	for i := 0; i < nQuestions; i++ {
		randomDomain := fmt.Sprintf(
			"%s.%s.",
			c.randomString(20),
			c.randomString(3),
		)
		m.SetQuestion(dns.CanonicalName(randomDomain), dns.TypeMX)
	}
	m.RecursionDesired = true
	r, rtt, err := c.client.Exchange(m, c.hostPort)
	if err != nil {
		return
	}
	dnsError = dns.RcodeToString[r.Rcode]
	return
}

func (c *DNSClient) randomString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
