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
	showSystemInfo(host, user, password)
}

func showSystemInfo(host, user, password string) {
	var (
		client *gosiklu.Client
		err    error
		reply  gosiklu.SikluData
	)
	if client, err = gosiklu.New(context.Background(), host, user, password); err != nil {
		log.Fatalf("Error creating client: %v", err)
	}
	defer client.Close()
	if reply, err = client.GetInfo([]string{"system", "event-cfg"}); err != nil {
		log.Fatalf("Error getting system info: %v", err)
	}
	log.Printf("System Info: %v", reply.GetInfoByName("system"))
	ec := reply.GetInfoByType("event-cfg")
	for _, info := range ec {
		for _, attr := range info.Attr {
			if attr.Name == "event-cfg-name" {
				log.Printf("\tEvent-Cfg-Name: %v\n", attr.Value)
			}
		}
	}
}
