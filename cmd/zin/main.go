package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"time"
	"zin-engine/engine"
	"zin-engine/utils"
)

const zinVersion = "zin/1.0"

func main() {

	// Define flags
	port := flag.String("p", "9001", "Port to listen on")
	rootDir := flag.String("r", "", "Root directory path")

	// Check if -r is not provided, try to set it to current working directory
	*rootDir = utils.GetCurrentWorkingDir(*rootDir)

	// Parse command-line flags
	flag.Parse()
	utils.PrintASCII(*port, *rootDir, zinVersion)

	// Start the engine
	address := ":" + *port
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("Error starting TCP server:\n %v", err)
		return
	}
	defer listener.Close()
	fmt.Printf("\n✅ Listening to requests...\n")

	// Handle Incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept: %v", err)
			continue
		}

		go func(conn net.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Send request to engine
			engine.HandleConnection(ctx, conn, *rootDir, zinVersion)
			fmt.Println("\n<<<<----- END")
		}(conn)
	}
}
