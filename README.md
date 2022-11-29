# lets-reach-consensus

## Usage
To run the consensus experiment, open one terminal to act as the controller as well as however many extra terminals you would like to use as nodes. For the controller, run `go run controller.go` which will start the controller process. For each node, run `go run node.go <port number> <input>`, where the port should be anything but 8080 (this is the fixed controller port) and the input should be 0 or 1. Once each node has displayed that it is connected to the host, begin the experiment by simply typing the fault tolerance you would like to use for the system `<faults>` into the controller terminal and press enter. This will trigger the experiment to begin simultaneously amongst all of the nodes. Typing `KILL` into the controller terminal will also kill the simulation at any time.

## Program Flow

### Controller
Using a variety of custom structs such as `message`, `sendable`, and `receivable` the controller accepts connections from all nodes who want to join the experiment. It saves a list of the ports of each joined node, and when the experiment starts as triggered by user input it sends the list of all the peer node ports (not including the receiving node's port) to each node in the system for its own use, as well as the fault tolerance. From here it waits for any kill signal from user input, otherwise continuing to run.

### Nodes
This process implements custom structs such as `message`, `sendable`, and `receivable` as well as `overview` and `round`. It also implements a high level abstraction, `addMessage()`, which allowed us to easily add incoming message data to the local store in each node. Once a node is started, it connects to the controller process and sends its own port number for use by other nodes during the simulation. It then waits for the initialization message from the controller, at which point it sets up unicast connections with all other nodes in the system based on the portlist it receives from the controller. From here, it runs the approximate consensus protocol, updating its local store as messages are sent and received in accordance to the algorithm. Once consensus is determined to have been achieved, the node exits the consensus protocol loop and logs the number of rounds and time needed to achieve consensus. It then kills the simulation for this node.

## Additional Notes

### MUTEX
The `overview` struct in `node.go` contains a slice of pointers to `round` structs. Because of this design choice the round in the overview is considered to be in its own address space. Because of this, both struct definitions have their own mutexes rather than sharing one. By having the overview mutex only be invoked when dereferencing a round and having each round have its own mutex, rounds can be more quickly accessed and different rounds can be modified at the same time. 

### Behavioral Observation
Given our implementation of approximate consensus and how we determine when consensus is achieved, there is the possibility that once a necessary amount of nodes reach consensus, it could cause other nodes to fail. This is due to the nodes that reached consensus stopping the broadcast of their state, which could potentially leave remaining nodes waiting. We don't see this as a road block in the correctness of our simulation as consensus is still properly reached amongst the required number of nodes. If we were inclined to change this behavior, however, we could change their behavior by either making hanging nodes timeout eventually or force nodes that have reached consensus to continue sending their state for a given number of rounds.
