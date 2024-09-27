package main

import (
	"context"
	"flag"
	"github.com/jjcinaz/gosiklu"
	"log"
	"os"
)

func main() {
	var (
		host, user, password string
	)
	flag.StringVar(&host, "host", "", "IP address or hostname of the radio")
	flag.StringVar(&user, "user", os.Getenv("SIKLU_USER"), "Username for the radio")
	flag.StringVar(&password, "password", os.Getenv("SIKLU_PASSWORD"), "Password for the radio")
	flag.Parse()
	if len(host) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}
	saveConfig(host, user, password)
}

func saveConfig(host, user, password string) {
	var (
		client *gosiklu.Client
		err    error
	)
	if client, err = gosiklu.New(context.Background(), host, user, password); err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()
	if err = client.SaveRunning(); err != nil {
		log.Fatalf("Error saving config: %v", err)
	}
	log.Printf("Saved Config")
}
