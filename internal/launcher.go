package internal

import (
	"log"
	"runtime"
	"sync"
	"time"
)

type Launcher struct {
	nRoutines  int
	nQuestions int
	hostPort   string
	sleepTime  *time.Duration
	stop       chan struct{}
	wg         sync.WaitGroup
}

func NewLauncher(nRoutines, nQuestions int, hostPort string, sleep *time.Duration) Launcher {
	return Launcher{
		nRoutines:  nRoutines,
		nQuestions: nQuestions,
		hostPort:   hostPort,
		sleepTime:  sleep,
		stop:       make(chan struct{}),
	}
}

func (l *Launcher) Start() {
	if l.nRoutines == 0 {
		l.nRoutines = runtime.NumCPU()
	}
	for i := 0; i < l.nRoutines; i++ {
		l.wg.Add(1)
		go l.runner()
	}
}

func (l *Launcher) runner() {
	client := NewDNSClient(l.hostPort)
	for {
		rtt, dnsErr, err := client.SendJunkDomainsRequest(l.nQuestions)
		errorMessage := "none"
		if err != nil {
			errorMessage = err.Error()
		}
		if dnsErr == "" {
			dnsErr = "none"
		}
		log.Printf("RTT: %s, DNS_ERR: %s, NET_ERR: %s", rtt.String(), dnsErr, errorMessage)
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

func (l *Launcher) Stop() {
	close(l.stop)
	l.wg.Wait()
}
