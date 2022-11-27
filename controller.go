package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

func handle_connections(source net.Listener, nodes *sync.Map) {
	for {
		c, err := source.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Handle incoming messages from client
		go func() {
			dec := gob.NewDecoder(c)
			var message string
			nodeport := ""
			for {
				err = dec.Decode(&message)
				if err != nil {
					return
				}

				// Add node encoder to nodes map on receiving first message, assuming it is the port from node
				if nodeport == "" {
					(*nodes).Store(message, gob.NewEncoder(c))
					nodeport = message
					fmt.Println("New Node: " + nodeport)
					continue
				}

				// Handle other messages from node
			}
		}()
	}
}

func main() {

	// Initialize receiver
	address := "127.0.0.1:8080" //+ port
	l, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer l.Close()

	var nodes sync.Map

	go handle_connections(l, &nodes)

	// Awaits user input to begin experiment
	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		text = strings.TrimSpace(text)

		faults, err := strconv.Atoi(text)
		if err != nil {
			fmt.Println("Please enter an integer for fault tolerance")
			continue
		}

		var nodeList []string

		// Create nodeList
		nodes.Range(func(key, value interface{}) bool {
			nodeList = append(nodeList, key.(string))
			return true
		})

		nodes.Range(func(key, value interface{}) bool {
			outgoingEnc := value.(*gob.Encoder)
			messageMap := make(map[string]interface{})
			messageMap["faults"] = faults
			messageMap["nodes"] = nodeList
			outgoingEnc.Encode(messageMap)
			return true
		})
	}
}
