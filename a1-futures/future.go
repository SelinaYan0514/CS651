// Package future provides a Future type that can be used to
// represent a value that will be available in the future.
package future

import (
	"net/rpc"
	"time"
)

// WeatherDataResult be used in GetWeatherData.
type WeatherDataResult struct {
	Value interface{}
	Err   error
}

// TemperatureRequest represents an RPC request with a station ID
type TemperatureRequest struct {
	StationID string
}

// TemperatureResponse represents an RPC response with the temperature value
type TemperatureResponse struct {
	Temperature float64
}

type Future struct {
	result chan interface{}
}

func NewFuture() *Future {
	return &Future{
		result: make(chan interface{}, 1),
	}
}

func (f *Future) CompleteFuture(res interface{}) {
	f.result <- res
	f.CloseFuture()
}

func (f *Future) GetResult() interface{} {
	return <-f.result
}

func (f *Future) CloseFuture() {
	close(f.result)
}

// Wait waits for the first n futures to return or for the timeout to expire,
// whichever happens first.
func Wait(futures []*Future, n int, timeout time.Duration, filter func(interface{}) bool) []interface{} {
	// TODO: Your code here
	return nil
}

// User Defined Function Logic

// GetWeatherData implementation which immediately returns a Future.
func GetWeatherData(client *rpc.Client, id int) *Future {
	// TODO: Your code here
	return nil
}

// heatWaveWarning is the filter function for the received weatherData.
// Should be used to keep only temperatures > 35 degrees Celsius.
func heatWaveWarning(res interface{}) bool {
	// TODO: Your code here
	return false
}
