package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
)

const DefaultLineHistory = 1000
const DefaultHost = "localhost"
const DefaultPort = 5055

func main() {
	defaultAddress := fmt.Sprintf("%s:%d", DefaultHost, DefaultPort)
	scanner := bufio.NewScanner(os.Stdin)

	filter := os.Args[1]
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
		if err := manager.IngestLine(line); err != nil {
			panic(err)
		}
	}
}
