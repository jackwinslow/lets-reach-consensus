package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
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

type round struct {
	mu       sync.Mutex
	min      float32
	max      float32
	average  float32
	num_rec  int
	received []message
}

type overview struct {
	mu     sync.Mutex
	Rounds map[int]*round
}

func initOverview() *overview {
	m := make(map[int]*round)
	return &overview{Rounds: m}
}

// adds the message to the overview, return the new difference and average of the messages round
func (o *overview) addMessage(m message) (diff float32, nr int, avg float32) {
	o.mu.Lock()
	if _, ok := o.Rounds[m.R]; !ok { // if round the round doesn't exist
		// map a round struct to the round number, initialize with message
		o.Rounds[m.R] = &round{min: m.V, max: m.V, average: m.V, num_rec: 1, received: []message{m}}
		o.mu.Unlock()
	} else {
		// update round values as necessary
		r := o.Rounds[m.R]
		o.mu.Unlock() // since r is a pointer, we no longer access overview and can release it, using round lock instead
		r.mu.Lock()
		if m.V < r.min {
			r.min = m.V
		}
		mn := r.min
		if m.V > r.max {
			r.max = m.V
		}
		mx := r.max
		r.average = (r.average*float32(r.num_rec) + m.V) / (float32(r.num_rec) + 1)
		av := r.average
		r.num_rec += 1
		nr := r.num_rec
		r.received = append(r.received, m)
		r.mu.Unlock()
		return (mx - mn), nr, av
	}
	return 0, 1, m.V // WARNING: diff is 0 if adding first message of round, should be handled
}

func main() {

	var state float32
	var n, f int
	var nodes []int
	round := 0
	ov := initOverview()

	port, err := strconv.Atoi(os.Args[1])
	tempState, err := strconv.ParseFloat(os.Args[2], 32)
	if err != nil {
		// do something sensible
	}
	state = float32(tempState)

	if err != nil {
		fmt.Println("Please provide a valid port")
		return
	}

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
	var start_message sendable
	decerr := dec.Decode(&start_message)
	if decerr != nil {
		fmt.Println(decerr)
	}

	nodes = start_message.Portlist
	n = len(nodes)
	f = start_message.Faults
	fmt.Println(nodes, n, f)

	// ------------------- SETUP UNICAST -------------------

	// assign node to listen to the port
	source_server := initialize_source(port)
	defer source_server.Close()

	// initialize an empty slice to store active outgoingConnections connections
	var encoders []gob.Encoder

	// activate reciever
	go unicast_recieve(source_server, ov)

	// Initalize outgoing connections with each node
	for _, node := range nodes {
		encoders = append(encoders, initialize_outgoing(node))
	}

	// -----------------------------------------------------

	// Each loop is one round
	for {
		round = round + 1
		// UNICAST TO EVERYONE ELSE INCLUDING SELF
		curr_message := message{
			V: state,
			R: round,
		}
		go func() {
			for _, encoder := range encoders {
				unicast_send(encoder, curr_message)
			}
		}()

		// Do waiting here
		for {
			if _, ok := ov.Rounds[round]; ok {
				if ov.Rounds[round].num_rec <= (n - f) {
					break
				}
			}
		}

		fmt.Print("Completed Round " + strconv.Itoa(round) + ":")
		fmt.Println(ov.Rounds[round].average)

		state = ov.Rounds[round].average

		time.Sleep(time.Duration(rand.Intn(1000)+1000) * time.Millisecond)

		if round > 10 {
			break
		}
	}

	fmt.Println("Reached consenus")
}

func initialize_source(port int) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}
	return ln
}

// assigns connections to individual reader goroutines that route messages into the proper channel
func unicast_recieve(source net.Listener, ov *overview) {
	for {

		// accept incoming connections
		conn, err := source.Accept()
		if err != nil {
			log.Fatal(err)
		}

		// pass connection into subproccess to handle incoming messages
		go func(conn net.Conn) {
			dec := gob.NewDecoder(conn)
			var message message
			for {
				dec.Decode(&message)
				ov.addMessage(message)
			}
		}(conn)

	}
}

func initialize_outgoing(port int) gob.Encoder {
	conn, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		log.Fatal(err)
	}

	enc := gob.NewEncoder(conn)

	return *enc
}

func unicast_send(enc gob.Encoder, message message) {
	err := enc.Encode(message)
	if err != nil {
		log.Fatal()
	}
}
