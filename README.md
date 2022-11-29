# lets-reach-consensus

### MUTEX
the `overview` struct contains a slice of pointers to `round` structs. Because of this design choice the round in the overview is considered to be in its own address space. Because of this, both struct definitions have their own mutexes rather than sharing one. By having the overview mutex only be invoked when dereferencing a round and having each round have its own mutex, rounds can be more quickly accessed and different rounds can be modified at the same time. 
