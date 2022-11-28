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

	ov := initOverview()

	// Channel used to trigger kill switch
	gold_chain := make(chan int)

	var faults int

	go func() {

		for {

			// accept incoming connections
			c, err := l.Accept()
			if err != nil {
				fmt.Println(err)
				return
			}

			go func() {

				// create decoder
				dec := gob.NewDecoder(c)
				var received receivable
				for {
					// listen for receivable message
					err := dec.Decode(&received)
					if err != nil {
						log.Fatal(err)
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

					case "STATE":
						// add state message to overview, if difference leq than threshold, output round & average
						if diff, m_in_r, average := ov.addMessage(received.State); diff <= 0.001 {
							if m_in_r >= len(nodes.Encoders)-faults {
								final_round := received.State.R
								gold_chain <- 1
								fmt.Println(final_round)
								fmt.Println(average)
							}
						}
					}

				}
			}()

		}
	}()

	defer l.Close()

	// Awaits user input to begin experiment
	initialize := true
	for initialize {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		text = strings.TrimSpace(text)

		conv, err := strconv.Atoi(text)
		faults := conv
		if err != nil {
			fmt.Println("Please enter an integer for fault tolerance")
			continue
		}

		start_message := sendable{
			Type:     "START",
			Portlist: ports.List,
			Faults:   faults,
		}

		// Create nodeList
		for _, node := range nodes.Encoders {
			node.Encode(start_message)
		}

		initialize = false
	}

	for {

	}

	// <-gold_chain
	// TIME TO DIE, send sendable{Type: "KILL"} to all nodes

}
