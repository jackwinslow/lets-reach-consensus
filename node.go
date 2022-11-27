package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
)

func main() {

	var n, f int
	var nodes []string

	port := os.Args[1]
	// state := os.Args[2]

	// Setup controller connection
	c, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Sends the controller port # upon connection initialization
	enc := gob.NewEncoder(c)
	enc.Encode(port)

	// Handles init message from controller
	dec := gob.NewDecoder(c)
	var message map[string]interface{}
	decerr := dec.Decode(&message)
	if decerr == nil {
		nodes = message["nodes"].([]string)
		n = len(nodes)
		f = message["faults"].(int)
		fmt.Println(nodes, n, f)
	}

	// round := 0
	// for {
	// 	// UNICAST TO EVERYONE ELSE INCLUDING SELF

	// 	// WAIT UNTIL RECEIVED N-F messages including self
	// 	avg := 0
	// 	count := 0

	// 	// Wait for n - f messages received

	// 	// Calculate average
	// 	for count < (n - f) {
	// 		avg = avg + // received value
	// 	}

	// 	avg = avg / (n - f)

	// 	state = avg

	// 	round = round + 1

	// 	// CHECK IF WE SHOULD KEEP GOING
	// 	if false {

	// 	} else {
	// 		break;
	// 	}
	// }

	// fmt.Println("Reached consenus")
}
