package concurrent

import (
	"container/heap"
	"time"
)

// Balancer is a type that can balance load among a set of workers
//
type Balancer struct {
	errorChan              chan error   // The channel used by all workers to report error
	requestHandledChan     chan *Worker // The channel used by a worker to signal balancer that a work has been executed
	tearDownChan           chan bool    // The channel that all workers listening for force quit signal
	workerFinishedChan     chan *Worker // The channel that all worker used to signal balancer that it exiting
	allWorkersFinishedChan chan bool    // The channel this balancer signals once all worker signals it's exit on workerFinishedChan
	pool                   Pool         // Pool of workers that this load balancer balances
	workerCount            int          // The number of workers
}

// The size of work channel associated with each worker this balancer manages.
//
const workerQueueSize int = 3

// NewBalancer creates a new instance of Balancer that needs to balance load between 'workerCount' workers
//
func NewBalancer(workerCount int) *Balancer {
	balancer := &Balancer{
		workerCount: workerCount,
		pool: Pool{
			Workers: make([]*Worker, workerCount),
		},
	}
	return balancer
}

// Init initializes all channels and start the workers.
//
func (b *Balancer) Init() {
	b.errorChan = make(chan error, 0)
	b.requestHandledChan = make(chan *Worker, 0)
	b.workerFinishedChan = make(chan *Worker, 0)
	b.allWorkersFinishedChan = make(chan bool, 0)
	b.tearDownChan = make(chan bool, 0)
	for i := 0; i < b.workerCount; i++ {
		b.pool.Workers[i] = NewWorker(i, workerQueueSize, &(b.pool), b.errorChan, b.requestHandledChan, b.workerFinishedChan)
		(b.pool.Workers[i]).Run(b.tearDownChan)
	}
}

// TearDownWorkers sends a force quit signal to all workers, which case worker to quit as soon as possible,
// workers won't drain it's request channel in this case.
//
func (b *Balancer) TearDownWorkers() {
	close(b.tearDownChan)
}

// Run read request from the request channel identified by the parameter requestChan and dispatch it the worker
// with least load. This method returns two channels, a channel to communicate error from any worker back to
// the consumer of balancer and second channel is used by the balancer to signal consumer that all workers has
// been finished executing.
//
func (b *Balancer) Run(requestChan <-chan *Request) (<-chan error, <-chan bool) {
	// Request dispatcher
	go func() {
		for {
			requestToHandle, ok := <-requestChan
			if !ok {
				b.closeWorkersRequestChannel()
				return
			}
			b.dispatch(requestToHandle)
		}
	}()

	// listener for worker status
	go func() {
		remainingWorkers := b.workerCount
		for {
			select {
			case w := <-b.requestHandledChan:
				b.completed(w)
			case _ = <-b.workerFinishedChan:
				remainingWorkers--
				if remainingWorkers == 0 {
					b.allWorkersFinishedChan <- true // All workers has been exited
					return
				}
			}
		}
	}()

	return b.errorChan, b.allWorkersFinishedChan
}

// closeWorkersRequestChannel closes the Request channel of all workers, this indicates that no
// more work will not be send the channel so that the workers can gracefully exit after handling
// any pending work in the channel.
//
func (b *Balancer) closeWorkersRequestChannel() {
	for i := 0; i < b.workerCount; i++ {
		close((b.pool.Workers[i]).RequestsToHandleChan)
	}
}

// dispatch dispatches the request to the worker with least load. If all workers are completely
// busy (i.e. there Pending request count is currently equal to the maximum load) then this
// method will poll until one worker is available.
//
func (b *Balancer) dispatch(request *Request) {
	for {
		if b.pool.Workers[0].Pending >= workerQueueSize {
			// Wait for a worker to be available
			time.Sleep(500 * time.Millisecond)
		} else {
			b.pool.Lock()
			worker := b.pool.Workers[0]
			worker.Pending++
			heap.Fix(&b.pool, worker.Index)
			worker.RequestsToHandleChan <- request
			b.pool.Unlock()
			return
		}
	}
}

// completed is called when a worker finishes one work, it updates the load status of the given the
// worker.
//
func (b *Balancer) completed(worker *Worker) {
	b.pool.Lock()
	worker.Pending--
	heap.Fix(&b.pool, worker.Index)
	b.pool.Unlock()
}

// WorkersCurrentLoad returns the load of the workers this balancer manages as comma separated string
// values where each value consists of worker id (Worker.Id property) and pending requests associated
// with the worker.
//
func (b *Balancer) WorkersCurrentLoad() string {
	return b.pool.WorkersCurrentLoad()
}
