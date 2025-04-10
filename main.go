package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
)

const DefaultLineHistory = 1000
const DefaultHost = "localhost"
const DefaultPort = 5055

func main() {
	port := flag.Int("port", DefaultPort, "port to listen on")
	host := flag.String("host", DefaultHost, "hostname to bind to")
	flag.Parse()

	defaultAddress := fmt.Sprintf("%s:%d", *host, *port)
	scanner := bufio.NewScanner(os.Stdin)

	filter := flag.Arg(0)
	manager, err := NewChannelManager(filter)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	BuildRoutes(mux, manager)

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
