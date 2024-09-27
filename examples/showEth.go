package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/goterm/term"
	"github.com/jjcinaz/gosiklu"
	"os"
)

func main() {
	var (
		host, user, password string
		fixup                bool
	)
	flag.StringVar(&host, "host", "", "IP address or hostname of the radio")
	flag.StringVar(&user, "user", os.Getenv("SIKLU_USER"), "Username for the radio")
	flag.StringVar(&password, "password", os.Getenv("SIKLU_PASSWORD"), "Password for the radio")
	flag.BoolVar(&fixup, "fixup", false, "Fixup the eth2/3 admin state")
	flag.Parse()
	if fixup {
		fixupEth(host, user, password)
	} else {
		showEthInfo(host, user, password)
	}
}

func fixupEth(host, user, password string) {
	var (
		client *gosiklu.Client
		err    error
		reply  gosiklu.SikluData
	)
	if client, err = gosiklu.New(context.Background(), host, user, password); err != nil {
		fmt.Println(term.Redf("unable to connect to Web server at %s err: %v", host, err))
		return
	}
	defer client.Close()
	if reply, err = client.GetInfo([]string{"system", "eth"}); err != nil {
		fmt.Println(term.Redf("unable to get info at %s err: %v", host, err))
		return
	}
	name := reply.GetAttrValue("system", "name")
	if reply.GetAttrValue("eth eth1", "operational") == "up" {
		cmds1 := make([]string, 0)
		if reply.GetAttrValue("eth eth2", "admin") == "up" && reply.GetAttrValue("eth eth2", "operational") == "down" {
			cmds1 = append(cmds1, "set eth eth2 admin down")
		}
		if reply.GetAttrValue("eth eth3", "admin") == "up" && reply.GetAttrValue("eth eth3", "operational") == "down" {
			cmds1 = append(cmds1, "set eth eth3 admin down")
		}
		if len(cmds1) > 0 {
			var cmdReply gosiklu.CommandReply
			cmdReply, err = client.Command(cmds1)
			if err != nil {
				fmt.Println(term.Redf("ERROR: Unable to update device %s at %s err: %v", name, host, err))
			} else if cmdReply.AllWorked() {
				fmt.Println(term.Bluef("INFO: Device %s at %s updated", name, host))
				saveIfNeeded(client)
			} else {
				for i, c := range cmdReply.EndCode {
					if c != "ok" {
						fmt.Println(term.Redf("ERROR: Device %s, IP %s command {%s} failed: %s", name, host, cmds1[i], cmdReply.Text[i]))
					}
				}
			}
		} else {
			fmt.Println(term.Greenf("INFO: Device %s at %s needs no update", name, host))
		}
	}
}

func saveIfNeeded(client *gosiklu.Client) {
	var (
		err   error
		reply gosiklu.SikluData
	)
	reply, err = client.GetInfo([]string{"system"})
	if err != nil {
		fmt.Println(term.Redf("ERROR: Unable to get system info from device at IP %s err: %v", client.Host, err))
		return
	}
	if saved, found := reply.GetAttr("system", "config-saved"); found {
		if saved == "false" {
			fmt.Print(term.Yellowf("INFO: Device at %s NEEDS a config save...", client.Host))
			if err = client.SaveRunning(); err != nil {
				fmt.Println(term.Redf("ERROR: %v", err))
			} else {
				fmt.Println(term.Greenf("Saved"))
			}
		} else {
			fmt.Println(term.Bluef("INFO: Device at %s doesn't need a config save", client.Host))
		}
	}
}

func showEthInfo(host, user, password string) {
	var (
		client *gosiklu.Client
		err    error
		reply  gosiklu.SikluData
	)
	fmt.Printf("%s", host)
	if client, err = gosiklu.New(context.Background(), host, user, password); err != nil {
		fmt.Printf(",\"%s\"\n", err)
		return
	}
	defer client.Close()
	if reply, err = client.GetInfo([]string{"system", "eth"}); err != nil {
		fmt.Printf(",\"%s\"\n", err)
		return
	}
	fmt.Printf(",\"%s\"", reply.GetAttrValue("system", "name"))
	fmt.Printf(",%s,%s", reply.GetAttrValue("eth eth1", "admin"), reply.GetAttrValue("eth eth1", "operational"))
	fmt.Printf(",%s,%s", reply.GetAttrValue("eth eth2", "admin"), reply.GetAttrValue("eth eth2", "operational"))
	if x, found := reply.GetAttr("eth eth3", "admin"); found {
		fmt.Printf(",%s,%s", x, reply.GetAttrValue("eth eth3", "operational"))
	} else {
		fmt.Printf(",,")
	}
	fmt.Println("")
}
