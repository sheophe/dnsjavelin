package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sheophe/dnsjavelin/internal"
)

func main() {
	hostPort := flag.String("a", "", "address in format host:port")
	nRoutines := flag.Int("c", 0, "number of parallel connections")
	nQuestions := flag.Int("n", 20, "number of DNS qeustion in one request")
	victimDomain := flag.String("d", "", "victim domain name")

	sleep := flag.Duration("s", 0, "sleep between requests")
	flag.Parse()

	if *hostPort == "" {
		log.Fatalf("host and port must be specified")
	}

	launcher := internal.NewLauncher(*nRoutines, *nQuestions, *hostPort, *victimDomain, sleep)

	launcher.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(
		sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	<-sigChan

	launcher.Stop()
}
