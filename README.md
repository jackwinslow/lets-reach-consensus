# lets-reach-consensus

### Usage
To run the consensus experiment, open one terminal to act as the controller as well as however many extra terminals you would like to use as nodes. For the controller, run `go run controller.go` which will start the controller process. For each node, run `go run node.go <port number> <input>`, where the port should be anything but 8080 and the input should be 0 or 1.

### MUTEX
the `overview` struct contains a slice of pointers to `round` structs. Because of this design choice the round in the overview is considered to be in its own address space. Because of this, both struct definitions have their own mutexes rather than sharing one. By having the overview mutex only be invoked when dereferencing a round and having each round have its own mutex, rounds can be more quickly accessed and different rounds can be modified at the same time. 
