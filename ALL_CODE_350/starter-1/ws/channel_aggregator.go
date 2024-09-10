package main
import (
    "time"
    "math"
)
// Channel-based aggregator that reports the global average temperature periodically
//
// Report the averagage temperature across all `k` weatherstations every `averagePeriod`
// seconds by sending a `WeatherReport` struct to the `out` channel. The aggregator should
// terminate upon receiving a singnal on the `quit` channel.
//
// Note! To receive credit, channelAggregator must not use mutexes.
func channelAggregator(
	k int,
	averagePeriod float64,
	getWeatherData func(int, int) WeatherReport,
	out chan WeatherReport,
	quit chan struct{},
) {
	// first I will use a time stamp to make the cycle around the averagePeriod
	stamp := time.NewTicker(time.Duration(averagePeriod * float64(time.Second)))
	defer stamp.Stop()
	batch :=0 //declare the batches for when time average period runs out 
	signal := make(chan struct{}) // adding signal for go helper function to get all the data
	go retrieveData(k, batch, getWeatherData, out, signal, quit) // new go routine so we dont wait for every single channel

	for { 
		select {
		case <-stamp.C: // the average time period has passed and is time to call a go routine to get the data of the batch
			// changing the approach and using sigan to change 
					close(signal)
						signal = make(chan struct{})
						batch++
						go retrieveData(k, batch, getWeatherData, out, signal, quit) // new go routine so we dont wait for every single channel
						case <- quit:
							return
							}
						}
					}
					// function that will retrieve the data in a go routinr mode so we dont keep waiting 
					func retrieveData(
						k int,
						batch int,
						getWeatherData func(int, int) WeatherReport,
						out chan WeatherReport,
						signal chan struct{},
						quit chan struct{},
					) {
						responseChan := make(chan WeatherReport, k)
					// loop for all the k stations
						for i := 0; i < k; i++ {
							go func(id int) {
								response := getWeatherData(id, batch)
								select {
								case responseChan <- response:
								case <-quit:
									return
								}
							}(i)
						}

						sum := 0.0
					    count := 0
						collecting := true
						// loop for the iterations and checking iof we got a valid response of a station or if we did not
						for collecting {
							select {
							case report := <-responseChan:
								if !math.IsNaN(report.Value) {
									sum += report.Value
									count++
								}
							case <-signal: // using closing of the signal channel as to indicate collection should stop
								collecting = false
							}
						}
				   // switch cass to handle if we got any responses 
				   // also managing the quit calls 
				   switch {
                case count > 0:
                    average := sum / float64(count)
                    select {
                    case out <- WeatherReport{Value: average, Batch: batch}:
                    case <-quit:
                        return
                    }
                default:
                    select {
                    case out <- WeatherReport{Value: math.NaN(), Batch: batch}:
                    case <-quit:
                        return
                    }
                }
			} // it works after a long week of trials