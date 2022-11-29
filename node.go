package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"math"
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
func (o *overview) addMessage(m message) {
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
		if m.V > r.max {
			r.max = m.V
		}
		r.average = (r.average*float32(r.num_rec) + m.V) / (float32(r.num_rec) + 1)
		r.num_rec += 1
		r.received = append(r.received, m)
		r.mu.Unlock()

	}
}

func main() {

	var state float32
	var n, f int
	var nodes []int
	gold_chain := make(chan int) // channel used to block main from completing execution, used as kill switch
	r := 0
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

	fmt.Println("Connected to Host!")

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

	// goroutine to wait for message to kill execution
	go func() {
		var death_message sendable
		for {
			decerr := dec.Decode(&death_message)
			if decerr != nil {
				fmt.Println(decerr)
			}
			if death_message.Type == "KILL" {
				gold_chain <- 1
			}
		}
	}()

	nodes = start_message.Portlist
	n = len(nodes) + 1
	f = start_message.Faults

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

	// This is the main portion of the consensus algorithm
	// Each loop is one round
	start_time := time.Now()
	go func() {
		consensus := false
		for {
			r = r + 1
			// UNICAST TO EVERYONE ELSE
			curr_message := message{
				V: state,
				R: r,
			}
			go func() {
				for _, encoder := range encoders {
					unicast_send(encoder, curr_message)
				}
			}()

			// Do waiting here
			ov.mu.Lock()
			cr, ok := ov.Rounds[r]
			if !ok {

				// Simulates sending message to self
				//ov.Rounds[r] = &round{min: curr_message.V, max: curr_message.V, average: curr_message.V, num_rec: 1, received: []message{curr_message}}
				ov.Rounds[r] = &round{min: float32(math.Inf(1)), max: float32(math.Inf(-1))}
				cr = ov.Rounds[r]
			}
			ov.mu.Unlock()
			for {
				cr.mu.Lock()
				if cr.num_rec >= (n - (f + 1)) {
					if curr_message.V < cr.min {
						cr.min = curr_message.V
					}
					// mn := cr.min
					if curr_message.V > cr.max {
						cr.max = curr_message.V
					}
					// mx := cr.max
					cr.average = (cr.average*float32(cr.num_rec) + curr_message.V) / (float32(cr.num_rec) + 1)
					// av := cr.average
					cr.num_rec += 1
					// nr := cr.num_rec
					cr.received = append(cr.received, curr_message)
					cr.mu.Unlock()
					// fmt.Println(cr.min, cr.max, cr.average, cr.num_rec, cr.received)
					if cr.max-cr.min <= 0.001 {
						// fmt.Println(cr.min, cr.max, cr.average, cr.num_rec, cr.received)
						consensus = true
					}
					break
				}
				cr.mu.Unlock()
			}

			fmt.Print("Completed Round " + strconv.Itoa(r) + ": ")
			fmt.Println(ov.Rounds[r].average)

			state = ov.Rounds[r].average

			time.Sleep(time.Duration(rand.Intn(10)+5) * time.Millisecond) // added random delay bound

			if consensus {
				break
			}
		}
		elapsed := time.Since(start_time)
		fmt.Println("Reached consenus")
		fmt.Print("Rounds: ")
		fmt.Print(r)
		fmt.Print(", Time: ")
		fmt.Println(elapsed)
		gold_chain <- 1
		// time.Sleep(time.Duration(rand.Intn(1000)+1000) * time.Millisecond)
	}()

	<-gold_chain // channel blocks execution which would terminate the program
	log.Fatal("KILLING SIMULATION")

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
		fmt.Println("Node no longer connected")
	}
}
