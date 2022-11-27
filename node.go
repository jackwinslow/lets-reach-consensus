package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
)

type message struct {
	v float32
	r int
}

type sendable struct {
	Type     string // "PORTLIST", "START", "KILL"
	Portlist []int
}

type receivable struct {
	Type  string // "PORT", "STATE"
	Port  int
	State message
}

func main() {

	var n, f int
	var nodes []int
	// round := 0

	port, err := strconv.Atoi(os.Args[1])

	if err != nil {
		fmt.Println("Please provide a valid port")
		return
	}
	// state := os.Args[2]

	// Setup controller connection
	c, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Sends the controller port # upon connection initialization
	enc := gob.NewEncoder(c)
	err = enc.Encode(receivable{Type: "PORT", Port: port})
	if err != nil {
		fmt.Println(err)
	}

	// Handles init message from controller
	dec := gob.NewDecoder(c)
	var message map[string]interface{}
	decerr := dec.Decode(&message)
	if decerr == nil {
		nodes = message["nodes"].([]int)
		n = len(nodes)
		f = message["faults"].(int)
		fmt.Println(nodes, n, f)
	}

	// ------------------- SETUP UNICAST -------------------

	// assign node to listen to the port
	source_server := initialize_source(port)
	defer source_server.Close()

	// initialize an empty slice to store active outgoing connections
	var outgoing []net.Conn

	// activate reciever
	go unicast_recieve(source_server)

	// Initalize outgoing connections with each node
	for _, node := range nodes {
		outgoing = append(outgoing, initialize_outgoing(node))
	}

	// -----------------------------------------------------

	// Each loop is one round
	// for {
	// 	// UNICAST TO EVERYONE ELSE INCLUDING SELF
	// 	go func() {
	// 		for _, node := range nodes {
	// 			unicast_send(outgoing[fields[1]], self+" "+message)
	// 		}
	// 	}()

	// 	// Do waiting here
	// 	for {

	// 	}

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

func initialize_source(port int) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}
	return ln
}

// assigns connections to individual reader goroutines that route messages into the proper channel
func unicast_recieve(source net.Listener) {
	for {

		// accept incoming connections
		conn, err := source.Accept()
		if err != nil {
			log.Fatal(err)
		}

		// pass connection into subproccess to handle incoming messages
		go func(conn net.Conn) {
			for {
				message, err := bufio.NewReader(conn).ReadString('\n')

				if err == io.EOF {
					conn.Close()
				}

				if err == nil {
					fmt.Println(message)
				}
			}
		}(conn)

	}
}

func initialize_outgoing(port int) net.Conn {
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func unicast_send(destination net.Conn, message string) {
	_, err := destination.Write([]byte(message + "\n"))
	if err != nil {
		log.Fatal()
	}
}
