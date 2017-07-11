package progress

import (
	"time"
)

// Status can be used by a collection of workers (reporters) to report the amount of work done when they need,
// Status compute the overall progress at regular interval and report it.
//
type Status struct {
	bytesProcessedCountChan chan int64
	doneChan                chan bool
	bytesProcessed          int64
	totalBytes              int64
	alreadyProcessedBytes   int64
	startTime               time.Time
	throughputStats         *ComputeStats
}

// Record type is used by the ProgressStatus to report the progress at regular interval.
//
type Record struct {
	PercentComplete              float64
	AverageThroughputMbPerSecond float64
	RemainingDuration            time.Duration
	BytesProcessed               int64
}

// oneMB is one MegaByte
//
const oneMB = float64(1048576)

// nanosecondsInOneSecond is 1 second expressed as nano-second unit
//
const nanosecondsInOneSecond = 1000 * 1000 * 1000

// NewStatus creates a new instance of Status. reporterCount is the number of concurrent goroutines that want to
// report processed bytes count, alreadyProcessedBytes is the bytes already processed if any, the parameter
// totalBytes is the total number of bytes that the reports will be process eventually, the parameter computeStats
// is used to calculate the running average.
//
func NewStatus(reportersCount int, alreadyProcessedBytes, totalBytes int64, computeStats *ComputeStats) *Status {
	return &Status{
		bytesProcessedCountChan: make(chan int64, reportersCount),
		doneChan:                make(chan bool, 0),
		totalBytes:              totalBytes,
		alreadyProcessedBytes:   alreadyProcessedBytes,
		startTime:               time.Now(),
		throughputStats:         computeStats,
	}
}

// ReportBytesProcessedCount method is used to report the number of bytes processed.
//
func (s *Status) ReportBytesProcessedCount(count int64) {
	s.bytesProcessedCountChan <- count
}

// Run starts counting the reported processed bytes count and compute the progress, this method returns a channel,
// the computed progress will be send to this channel in regular interval. Once done with using ProgressStatus
// instance, you must call Dispose method otherwise there will be go routine leak.
//
func (s *Status) Run() <-chan *Record {
	go s.bytesProcessedCountReceiver()
	var outChan = make(chan *Record, 0)
	go s.progressRecordSender(outChan)
	return outChan
}

// Close disposes this ProgressStatus instance, an attempt to invoke ReportBytesProcessedCount method on a closed
// instance will be panic. Close also stops sending progress to the channel returned by Run method. Not calling
// Close will cause goroutine leak.
//
func (s *Status) Close() {
	close(s.bytesProcessedCountChan)
}

// bytesProcessedCountReceiver read the channel containing the collection of reported bytes count and update the total
// bytes processed. This method signal doneChan when there is no more data to read.
//
func (s *Status) bytesProcessedCountReceiver() {
	for c := range s.bytesProcessedCountChan {
		s.bytesProcessed += c
	}
	s.doneChan <- true
}

// progressRecordSender compute the progress information at regular interval and send it to channel outChan which is
// returned by the Run method
//
func (s *Status) progressRecordSender(outChan chan<- *Record) {
	progressRecord := &Record{}
	tickerChan := time.NewTicker(500 * time.Millisecond)
Loop:
	for {
		select {
		case <-tickerChan.C:
			computeAvg := s.throughputStats.ComputeAvg(s.throughputMBs())
			avtThroughputMbps := 8.0 * computeAvg
			remainingSeconds := (s.remainingMB() / computeAvg)

			progressRecord.PercentComplete = s.percentComplete()
			progressRecord.RemainingDuration = time.Duration(nanosecondsInOneSecond * remainingSeconds)
			progressRecord.AverageThroughputMbPerSecond = avtThroughputMbps
			progressRecord.BytesProcessed = s.bytesProcessed

			outChan <- progressRecord
		case <-s.doneChan:
			tickerChan.Stop()
			break Loop
		}
	}
	close(outChan)
}

// remainingMB returns remaining bytes to be processed as MB.
//
func (s *Status) remainingMB() float64 {
	return float64(s.totalBytes-s.bytesProcessed) / oneMB
}

// percentComplete returns the percentage of bytes processed out of total bytes.
//
func (s *Status) percentComplete() float64 {
	return float64(100.0) * (float64(s.bytesProcessed) / float64(s.totalBytes))
}

// processTime returns the Duration representing the time taken to process the bytes so far.
//
func (s *Status) processTime() time.Duration {
	return time.Since(s.startTime)
}

// throughputMBs returns the throughput in MB
//
func (s *Status) throughputMBs() float64 {
	return float64(s.bytesProcessed) / oneMB / s.processTime().Seconds()
}
