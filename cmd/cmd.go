package cmd

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type endpointArray []string

func (i *endpointArray) String() string {
	return ""
}

func (i *endpointArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	Endpoint endpointArray
	Timeout  time.Duration
	UseTui   bool
)

func LoadFlags() {
	flag.Var(&Endpoint, "endpoint", "The endpoint to check eg:\n - https://www.mywebsite.org\n - tcp://localhost:3306\n You can specify this flag multiple times.")
	flag.DurationVar(&Timeout, "timeout", 30*time.Second, "TLS connection timeout")
	flag.BoolVar(&UseTui, "tui", false, "Use the tui instead of the cli")
	flag.Parse()

	if len(Endpoint) == 0 {
		fmt.Println("You must specify at least one endpoint")
		os.Exit(1)
	}

}
