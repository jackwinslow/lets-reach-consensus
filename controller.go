package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
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
	if _, ok := o.Rounds[m.r]; !ok { // if round the round doesn't exist
		// map a round struct to the round number, initialize with message
		o.Rounds[m.r] = &round{min: m.v, max: m.v, average: m.v, num_rec: 1, received: []message{m}}
		o.mu.Unlock()
	} else {
		// update round values as necessary
		r := o.Rounds[m.r]
		o.mu.Unlock() // since r is a pointer, we no longer access overview and can release it, using round lock instead
		r.mu.Lock()
		if m.v < r.min {
			r.min = m.v
		}
		mn := r.min
		if m.v > r.max {
			r.max = m.v
		}
		mx := r.max
		r.average = (r.average*float32(r.num_rec) + m.v) / (float32(r.num_rec) + 1)
		av := r.average
		r.num_rec += 1
		nr := r.num_rec
		r.received = append(r.received, m)
		r.mu.Unlock()
		return (mx - mn), nr, av
	}
	return 0, 1, m.v // WARNING: diff is 0 if adding first message of round, should be handled
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

	gold_chain := make(chan int)

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

					fmt.Println(received)

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
							if m_in_r >= 5 { // 5 is placeholder, should be number of non-failed nodes
								final_round := received.State.r
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

	<-gold_chain
	// TIME TO DIE, send sendable{Type: "KILL"} to all nodes

}
