package main

import(
"math"
"sync"
"time"
)
	
// Mutex-based aggregator that reports the global average temperature periodically
//
// Report the averagage temperature across all `k` weatherstations every `averagePeriod`
// seconds by sending a `WeatherReport` struct to the `out` channel. The aggregator should
// terminate upon receiving a singnal on the `quit` channel.
//
// Note! To receive credit, mutexAggregator must implement a mutex based solution.
func mutexAggregator(
	k int,
	averagePeriod float64,
	getWeatherData func(int, int) WeatherReport,
	out chan WeatherReport,
	quit chan struct{},
) {
	// declared all the variables im using from the top of the function
	var (
		mutex sync.Mutex
		sum float64
		count int
		batch int
	)
	// same as in the other file, creating the ticker and ist defer statement
	stamp := time.NewTicker(time.Duration(averagePeriod * float64(time.Second)))
	defer stamp.Stop()

		// go function to calculate all the averages without wait time
				retrieveData := func(id, currentBatch int) {
				report := getWeatherData(id, currentBatch)
				mutex.Lock()
				// checking if the report belongs to the current batch and is valid
				switch{
				case batch == currentBatch && !math.IsNaN(report.Value):
					sum += report.Value
					count++
				default:
					sum += 0
					count += 0
				}
				mutex.Unlock()
				// end of the cases
			}
			// start a new batch of data collection
			newBatch := func(batch int) {
				// rest of variable declarations
			mutex.Lock()
            // Prepare for a new batch
            sum = 0.0
            count = 0
            mutex.Unlock()
				for i := 0; i < k; i++ {
					go retrieveData(i, batch)
				}
			}
			newBatch(batch)
			// batch 0
	for { 
		select {
		case <- quit:
			return // return statment in case we get an empty struct signal 
		case <- stamp.C: // the averagePeriod has been completed, time to move into a go routine and implement mutex
			// implement the mutex strategy
			// see if we compute or no
			mutex.Lock()
			switch {
			case count > 0:
				average := sum / float64(count)
				out <- WeatherReport{Value: average, Batch: batch}
			default:
				out <- WeatherReport{Value: math.NaN(), Batch: batch}
			}
			batch++
			mutex.Unlock()
			// call loop
			newBatch(batch)
			}
			
		}
	}
