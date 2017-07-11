package concurrent

import "fmt"

// Worker represents a type which can listen for work from a channel and run them
//
type Worker struct {
	RequestsToHandleChan chan *Request  // The buffered channel of works this worker needs to handle
	Pending              int            // The number of pending requests this worker needs to handle (i.e. worker load)
	errorChan            chan<- error   // The channel to report failure in executing work
	requestHandledChan   chan<- *Worker // The channel to report that a work is done (irrespective of success or failure)
	workerFinishedChan   chan<- *Worker // The channel to signal that worker has finished (worker go-routine exited)
	ID                   int            // Unique Id for worker (Debugging purpose)
	Index                int            // The index of the item in the heap.
	pool                 *Pool          // The parent pool holding all workers (used for work stealing)
}

// The maximum number of times a work needs to be retried before reporting failure on errorChan.
//
const maxRetryCount int = 5

// NewWorker creates a new instance of the worker with the given work channel size.
// errorChan is the channel to report the failure in addressing a work request after all
// retries, each time a work is completed (failure or success) doneChan will be signalled
//
func NewWorker(id int, workChannelSize int, pool *Pool, errorChan chan<- error, requestHandledChan chan<- *Worker, workerFinishedChan chan<- *Worker) *Worker {
	return &Worker{
		ID:                   id,
		RequestsToHandleChan: make(chan *Request, workChannelSize),
		errorChan:            errorChan,
		requestHandledChan:   requestHandledChan,
		workerFinishedChan:   workerFinishedChan,
		pool:                 pool,
	}
}

// Run starts a go-routine that read work from work-queue associated with the worker and executes one
// at a time. The go-routine returns/exit once one of the following condition is met:
//   1. The work-queue is closed and drained and there is no work to steal from peers worker's work-queue
//   2. A signal is received in the tearDownChan channel parameter
//
// After executing each work, this method sends report to Worker::requestHandledChan channel
// If a work fails after maximum retry, this method sends report to Worker::errorChan channel
//
func (w *Worker) Run(tearDownChan <-chan bool) {
	go func() {
		defer func() {
			// Signal balancer that worker is finished
			w.workerFinishedChan <- w
		}()

		var requestToHandle *Request
		var ok bool
		for {
			select {
			case requestToHandle, ok = <-w.RequestsToHandleChan:
				if !ok {
					// Request channel is closed and drained, worker can try to steal work from others.
					//
					// Note: load balancer does not play any role in stealing, load balancer closes send-end
					// of all worker queue's at the same time, at this point we are sure that no more new job
					// will be scheduled. Once we start stealing "Worker::Pending" won't reflect correct load.
					requestToHandle = w.tryStealWork()
					if requestToHandle == nil {
						// Could not steal then return
						return
					}
				}
			case <-tearDownChan:
				// immediate stop, no need to drain the request channel
				return
			}

			var err error
			// Do work, retry on failure.
		Loop:
			for count := 0; count < maxRetryCount+1; count++ {
				select {
				case <-tearDownChan:
					return
				default:
					err = requestToHandle.Work() // Run work
					if err == nil || !requestToHandle.ShouldRetry(err) {
						break Loop
					}
				}
			}

			if err != nil {
				select {
				case w.errorChan <- fmt.Errorf("%s: %v", requestToHandle.ID, err):
				case <-tearDownChan:
					return
				}
			}

			select {
			case w.requestHandledChan <- w: // One work finished (successfully or unsuccessfully)
			case <-tearDownChan:
				return
			}
		}
	}()
}

// tryStealWork will try to steal a work from peer worker if available. If all peer channels are
// empty then return nil
//
func (w *Worker) tryStealWork() *Request {
	for _, w1 := range w.pool.Workers {
		request, ok := <-w1.RequestsToHandleChan
		if ok {
			return request
		}
	}
	return nil
}
