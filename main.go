package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const DefaultLineHistory = 1000
const DefaultHost = "localhost"
const DefaultPort = 5055
const FilterFieldDelimiter = ":"

func main() {
	port := flag.Int("port", DefaultPort, "port to listen on")
	host := flag.String("host", DefaultHost, "hostname to bind to")
	devMode := flag.Bool("dev", false, "enable developer mode")
	flag.Parse()

	defaultAddress := fmt.Sprintf("%s:%d", *host, *port)
	scanner := bufio.NewScanner(os.Stdin)

	filter := flag.Arg(0)
	manager, err := NewChannelManager(filter)
	if err != nil {
		panic(err)
	}

	for i, arg := range flag.Args() {
		if i == 0 {
			// Already captured
			continue
		}
		// Name of filter and filter regex separated by FilterFieldDelimiter.
		// If there is no FilterFieldDelimiter, name and filter are the same
		strs := strings.SplitN(arg, FilterFieldDelimiter, 2)
		name := strs[0]
		filter := name
		if len(strs) > 1 {
			filter = strs[1]
		}
		channel := NewChannel(name, nil, "")
		if err := channel.SetFilter(filter); err != nil {
			fmt.Printf("failed to parse arg filter %q: %s\n", filter, err)
			os.Exit(1)
		}
		if err := manager.AddChannel(channel); err != nil {
			panic(err)
		}
	}

	mux := http.NewServeMux()
	BuildRoutes(mux, manager, *devMode)

	go func() {
		panic(http.ListenAndServe(defaultAddress, mux))
	}()

	for scanner.Scan() {
		line := scanner.Text()
		if err := manager.IngestString(line); err != nil {
			panic(err)
		}
	}
}
