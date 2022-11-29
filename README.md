# lets-reach-consensus

### MUTEX
the `overview` struct contains a slice of pointers to `round` structs. Because of this design choice the rounds in the overview are in considered to be in their own address space, which means that changes to a round does not effect the overview its contained in. Because of this, both struct definitions have their own mutexes. By having the overview mutex only being invoked when dereferencing a round, rounds can be more quickly accessed, and different rounds can be modified at the same time. 
