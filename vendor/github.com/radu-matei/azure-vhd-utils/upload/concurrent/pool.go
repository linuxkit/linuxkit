package concurrent

import (
	"bytes"
	"fmt"
	"sync"
)

// Pool is the collection of Worker, it is a min-heap that implements heap.Interface.
// The priority is the number of pending works assigned to the worker. Lower the pending
// work count higher the priority. Pool embeds sync.RWMutex to support concurrent heap
// operation.
//
type Pool struct {
	sync.RWMutex           // If consumer want to use workers in a concurrent environment
	Workers      []*Worker // The workers
}

// Len returns number of workers in the pool.
//
func (p *Pool) Len() int {
	return len(p.Workers)
}

// Less returns true if priority of Worker instance at index i is less than priority of Worker
// instance at j, lower the pending value higher the priority
//
func (p *Pool) Less(i, j int) bool {
	return p.Workers[i].Pending < p.Workers[j].Pending
}

// Swap swaps the Worker instances at the given indices i and j
//
func (p Pool) Swap(i, j int) {
	p.Workers[i], p.Workers[j] = p.Workers[j], p.Workers[i]
	p.Workers[i].Index = i
	p.Workers[j].Index = j
}

// Push is used by heap.Push implementation, to add a worker w to a Pool pool, we call
// heap.Push(&pool, w) which invokes this method to add the worker to the end of collection
// then it fix the heap by moving the added item to its correct position.
//
func (p *Pool) Push(x interface{}) {
	n := len(p.Workers)
	worker := x.(*Worker)
	worker.Index = n
	(*p).Workers = append((*p).Workers, worker)
}

// Pop is used by heap.Pop implementation, to pop a worker w with minimum priority from a Pool
// p, we call w := heap.Pop(&p).(*Worker), which swap the min priority worker at the beginning
// of the pool with the end of item, fix the heap and then invokes this method for popping the
// worker from the end.
//
func (p *Pool) Pop() interface{} {
	n := len(p.Workers)
	w := (*p).Workers[n-1]
	w.Index = -1
	(*p).Workers = (*p).Workers[0 : n-1]
	return w
}

// WorkersCurrentLoad returns the load of the workers as comma separated string values, where
// each value consists of worker id (Worker.Id property) and pending requests associated with
// the worker.
//
func (p *Pool) WorkersCurrentLoad() string {
	var buffer bytes.Buffer
	buffer.WriteString("Load [")
	for _, w := range p.Workers {
		buffer.WriteString(fmt.Sprintf("%d:%d, ", w.ID, w.Pending))
	}
	buffer.WriteString("]")
	return buffer.String()
}
