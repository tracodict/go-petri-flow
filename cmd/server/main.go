package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-petri-flow/internal/api"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "Port to run the server on")
	flag.Parse()

	// Create API server
	server := api.NewServer()
	defer server.Close()

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down server...")
		server.Close()
		os.Exit(0)
	}()

	// Start the server
	log.Printf("Go Petri Flow server starting...")
	if err := server.StartServer(*port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

