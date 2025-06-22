package main

import (
	"context"
	"fmt"
	"net"
	"time"
	"zin-engine/engine"
)

func main() {

	port := ":9001"
	listener, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("Error starting TCP server: %s | File: %s", err.Error(), "main.go")
		return
	}
	defer listener.Close()

	fmt.Println("\n ")
	fmt.Println("  ______  _____   _   _  ")
	fmt.Println(" |___  / |_   _| | \\ | | ")
	fmt.Println("    / /    | |   |  \\| | ")
	fmt.Println("   / /     | |   | . ` | ")
	fmt.Println("  / /__   _| |_  | |\\  | ")
	fmt.Println(" /_____| |_____| |_| \\_| ")
	fmt.Println("• Version : zin 1.0")
	fmt.Println("• Port    : 9001")
	fmt.Println("• Local   : http://127.0.0.1:9001")
	fmt.Println("\n✅ Listening to requests...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %s | File: %s", err.Error(), "main.go")
			continue
		}
		go func(conn net.Conn) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			done := make(chan struct{})

			go func() {
				engine.HandleConnection(ctx, conn)
				close(done)
			}()

			select {
			case <-ctx.Done():
				fmt.Println("\n<<<<----- TIMEOUT-EXIT")
				conn.Close()
			case <-done:
				// connection handled successfully
				fmt.Println("\n<<<<----- END")
			}
		}(conn)
	}
}
