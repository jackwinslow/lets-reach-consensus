package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"log"

	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type message struct {
	V float32
	R int
}

type sendable struct {
	Type     string // "START", "KILL"
	Portlist []int
	Faults   int
}

type receivable struct {
	Type  string // "PORT", "STATE"
	Port  int
	State message
}

type p struct {
	mu   sync.Mutex
	List []int
}

type n struct {
	mu       sync.Mutex
	Encoders map[int]*gob.Encoder
}

func main() {

	// Initialize receiver
	address := "127.0.0.1:8080"
	l, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println(err)
		return
	}

	var list []int
	ports := p{List: list}

	encoders := make(map[int]*gob.Encoder)
	nodes := n{Encoders: encoders}

	go func() {

		for {

			// accept incoming connections
			c, err := l.Accept()
			if err != nil {
				return // silent suicide is performed since
			}

			go func() {

				// create decoder
				dec := gob.NewDecoder(c)
				var received receivable
				for {
					// listen for receivable message
					err := dec.Decode(&received)
					if err != nil {
						if err.Error() == "EOF" {
							log.Println("A node has reached consensus")
							return
						}
					}

					switch received.Type {
					// if receivable is a port message
					case "PORT":
						// append port to ports list
						ports.mu.Lock()
						nodes.mu.Lock()
						ports.List = append(ports.List, received.Port)
						nodes.Encoders[received.Port] = gob.NewEncoder(c)
						nodes.mu.Unlock()
						ports.mu.Unlock()
					}
				}
			}()

		}
	}()

	// Awaits user input to begin experiment
	initialize := true
	reader := bufio.NewReader(os.Stdin)
	for initialize {
		text, _ := reader.ReadString('\n')

		text = strings.TrimSpace(text)

		conv, err := strconv.Atoi(text)
		faults := conv
		if err != nil {
			fmt.Println("Please enter an integer for fault tolerance")
			continue
		}

		start_message := sendable{
			Type:   "START",
			Faults: faults,
		}

		// Sending portlist
		for np, node := range nodes.Encoders {
			// remove np from ports.List
			sending := []int{}
			for _, port := range ports.List {
				if port != np {
					sending = append(sending, port)
				}
			}
			start_message.Portlist = sending

			node.Encode(start_message)
		}

		initialize = false
	}

	// wait for "KILL" to be written in stdin to send kill signal to all nodes
	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "KILL" {
			log.Println("KILLING SIMULATION")
			for _, node := range nodes.Encoders {
				err = node.Encode(sendable{Type: "KILL"})
				if err != nil {continue}
			}
			break
		}
	}
	
	l.Close()


}