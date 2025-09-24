package future

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"net/rpc"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

var (
	mockServer  *httptest.Server
	mockHandler atomic.Value
)

// TestMain sets up the mock server and runs the tests.
func TestMain(m *testing.M) {
	// Start the mock server with a dynamic handler.
	mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Load the current handler and serve the request.
		handler := mockHandler.Load().(http.HandlerFunc)
		handler.ServeHTTP(w, r)
	}))

	// Run the tests.
	code := m.Run()

	// Clean up the mock server.
	defer mockServer.Close()

	// Exit with the code from m.Run().
	os.Exit(code)
}

func (m *WeatherService) GetTemperature(req TemperatureRequest, res *TemperatureResponse) error {
	return m.HandleTemperatureRequest(req, res)
}

type WeatherService struct {
	HandleTemperatureRequest func(req TemperatureRequest, res *TemperatureResponse) error
}

func defaultTemperatureHandler(req TemperatureRequest, res *TemperatureResponse) error {
	id, err := strconv.Atoi(req.StationID)
	if err != nil {
		return errors.New("invalid station ID")
	}

	time.Sleep(200 * time.Millisecond)

	if id == 1 {
		select {}
	}

	res.Temperature = float64(id) * 2.0
	return nil
}

func customTemperatureHandlerDelay(req TemperatureRequest, res *TemperatureResponse) error {
	id, err := strconv.Atoi(req.StationID)
	if err != nil {
		return errors.New("invalid station ID")
	}

	time.Sleep(200 * time.Millisecond)

	if id > 7 {
		time.Sleep(400 * time.Millisecond)
	}

	res.Temperature = float64(id) * 2.0
	return nil
}

// slowFunction simulates sending a slow function that immediately returns a Future
// and keeps running in the background until a result is available.
// Returns a random number between 0 and 10000 (inclusive).
func slowFunction(straggler bool, unreliable bool) *Future {
	f := NewFuture()
	go func() {
		// Simulate an asynchronous operation.
		time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

		if straggler {
			// Sleeps for longer, but will eventually complete.
			time.Sleep(time.Duration(100+rand.Intn(500)) * time.Millisecond)
		}

		if unreliable {
			// The future will never complete.
			// We must close the channel to return a <nil>,
			// else the future will never return anything.
			close(f.result)
			return
		}

		// Get random number.
		randomNumber := rand.Intn(10000)
		f.CompleteFuture(randomNumber)
	}()
	return f
}

func isEven(res interface{}) bool {
	return res.(int)%2 == 0
}

func startMockRPCServer(shutdownChan chan struct{}, port int, handler func(req TemperatureRequest, res *TemperatureResponse) error) {
	// Create a new RPC server instance
	rpcServer := rpc.NewServer()

	mockService := &WeatherService{
		HandleTemperatureRequest: handler,
	}

	rpcServer.Register(mockService)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-shutdownChan:
					return
				default:
					continue
				}
			}
			go rpcServer.ServeConn(conn)
		}
	}()

	// Wait for a shutdown signal
	<-shutdownChan
	listener.Close()
}

// getTestRPCClient connects to the mock RPC server.
func getTestRPCClient(port int) (*rpc.Client, error) {
	time.Sleep(1 * time.Second)
	return rpc.Dial("tcp", fmt.Sprintf("localhost:%d", port))
}

// TestFutureBasic tests if a minimum number of futures are completed
// with a very lenient timeout.
func TestFutureBasic(t *testing.T) {
	results := make([]*Future, 0)
	nPeers := 10
	requiredCompletedFutures := (nPeers / 2) + 1
	for i := 0; i < nPeers; i++ {
		results = append(results, slowFunction(false, false))
	}

	returnedResults := Wait(results, (nPeers/2)+1, 300*time.Millisecond, nil)

	if len(returnedResults) == 0 {
		t.Errorf("ERROR: Wait() returned an empty slice %v", returnedResults)
		return
	}

	if len(returnedResults) != requiredCompletedFutures {
		t.Errorf("ERROR: Completed Futures %v ( != %v)", len(returnedResults), requiredCompletedFutures)
	}
}

// TestFutureTimeout tests if the Wait function correctly handles
// the scenario where the timeout expires before enough futures complete.
func TestFutureTimeout(t *testing.T) {
	results := make([]*Future, 0)
	nPeers := 10                                                     // 10
	requiredCompletedFutures := (nPeers / 2) + 1                     // 6
	stragglerRequestCount := (nPeers - requiredCompletedFutures) + 2 // 6
	maximumFuturesToComplete := nPeers - stragglerRequestCount       // 4
	// while stragglerRequestCount >= 0, slowFunction call would fail.
	for i := 0; i < nPeers; i++ {
		results = append(results, slowFunction(stragglerRequestCount >= 0, false))
		stragglerRequestCount -= 1
	}

	returnedResults := Wait(results, (nPeers/2)+1, 300*time.Millisecond, nil)

	if len(returnedResults) == 0 {
		t.Errorf("ERROR: Wait() returned an empty slice %v", returnedResults)
		return
	}

	if len(returnedResults) > maximumFuturesToComplete {
		t.Errorf("ERROR: Completed Futures %v, expected at most %v", len(returnedResults), maximumFuturesToComplete)
	}
}

// TestFutureFilter tests if the Wait function correctly applies
// the post-completion logic and only includes futures that satisfy the logic.
func TestFutureFilter(t *testing.T) {
	results := make([]*Future, 0)
	nPeers := 10
	for i := 0; i < nPeers; i++ {
		results = append(results, slowFunction(false, false))
	}

	returnedResults := Wait(results, (nPeers/2)+1, 300*time.Millisecond, isEven)

	if len(returnedResults) == 0 {
		t.Errorf("ERROR: Wait() returned an empty slice %v", returnedResults)
		return
	}

	for i := 0; i < len(returnedResults); i++ {
		if returnedResults[i] == false {
			t.Errorf("ERROR: Expected only true values, got false at index %v %+v", i, returnedResults)
			return
		}
	}
}

// TestFutureUnreliable tests if the Wait function
// can handle futures that never complete.
func TestFutureUnreliable(t *testing.T) {
	results := make([]*Future, 0)
	nPeers := 10
	requiredCompletedFutures := (nPeers / 2) + 1
	for i := 0; i < nPeers; i++ {
		results = append(results, slowFunction(false, true))
	}

	returnedResults := Wait(results, (nPeers/2)+1, 300*time.Millisecond, nil)

	if len(returnedResults) == 0 {
		t.Errorf("ERROR: Wait() returned an empty slice %v", returnedResults)
		return
	}

	if len(returnedResults) != requiredCompletedFutures {
		t.Errorf("ERROR: Completed Futures %v ( != %v)", len(returnedResults), requiredCompletedFutures)
	}
	// fmt.Printf("Returned Values %+v \n", returnedResults)
}

// Tests for GetWeatherData

// TestGetWeatherDataBasic tests if GetWeatherData() is
// implemented as expected and returns valid Future objects.
func TestGetWeatherDataBasic(t *testing.T) {
	port := 51000
	shutdownChan := make(chan struct{})
	go startMockRPCServer(shutdownChan, port, defaultTemperatureHandler)
	defer close(shutdownChan)

	client, err := getTestRPCClient(port)
	if err != nil {
		t.Fatalf("Failed to connect to test RPC server: %v", err)
	}
	defer client.Close()

	results := make([]*Future, 0)
	idOffset := 2
	nPeers := 10                       // 10
	requiredCompletedFutures := nPeers // 10

	for i := 0; i < nPeers; i++ {
		// to start from id = idOffset
		results = append(results, GetWeatherData(client, i+idOffset))
	}

	weatherDataResults := Wait(results, requiredCompletedFutures, 300*time.Millisecond, nil)

	if len(weatherDataResults) == 0 {
		t.Errorf("ERROR: Wait() returned an empty slice %v", weatherDataResults)
		return
	}

	if len(weatherDataResults) != requiredCompletedFutures {
		t.Errorf("ERROR: Completed Futures %v ( != %v)", len(weatherDataResults), requiredCompletedFutures)
	}
}

// TestGetWeatherDataDelayedResponses tests that the correct
// number of futures is returned if the timeout expires
// before all futures complete.
func TestGetWeatherDataDelayedResponses(t *testing.T) {
	port := 51001
	shutdownChan := make(chan struct{})
	go startMockRPCServer(shutdownChan, port, customTemperatureHandlerDelay)
	defer close(shutdownChan)

	client, err := getTestRPCClient(port)
	if err != nil {
		t.Fatalf("Failed to connect to test RPC server: %v", err)
	}
	defer client.Close()

	results := make([]*Future, 0)
	idOffset := 2
	nPeers := 10                             // 10
	requiredCompletedFutures := 7 - idOffset // 7

	for i := 0; i < nPeers; i++ {
		// to start from id = idOffset
		results = append(results, GetWeatherData(client, i+idOffset))
	}

	weatherDataResults := Wait(results, requiredCompletedFutures, 300*time.Millisecond, nil)

	if len(weatherDataResults) == 0 {
		t.Errorf("ERROR: Wait() returned an empty slice %v", weatherDataResults)
		return
	}

	if len(weatherDataResults) != requiredCompletedFutures {
		t.Errorf("ERROR: Completed Futures %v ( != %v)", len(weatherDataResults), requiredCompletedFutures)
	}
}

// TestGetWeatherDataOneFail tests that the correct
// number of futures is returned if some (one) future never completes.
func TestGetWeatherDataOneFail(t *testing.T) {
	port := 51002
	shutdownChan := make(chan struct{})
	go startMockRPCServer(shutdownChan, port, defaultTemperatureHandler)
	defer close(shutdownChan)

	client, err := getTestRPCClient(port)
	if err != nil {
		t.Fatalf("Failed to connect to test RPC server: %v", err)
	}
	defer client.Close()

	results := make([]*Future, 0)
	idOffset := 1
	nPeers := 10                                  // 10
	requiredCompletedFutures := nPeers - idOffset // 9

	for i := 0; i < nPeers; i++ {
		// to start from id = idOffset
		results = append(results, GetWeatherData(client, i+idOffset))
	}

	weatherDataResults := Wait(results, requiredCompletedFutures, 2*time.Second, nil)
	if len(weatherDataResults) == 0 {
		t.Errorf("ERROR: Wait() returned an empty slice %v", weatherDataResults)
		return
	}
	if len(weatherDataResults) != requiredCompletedFutures {
		t.Errorf("ERROR: Completed Futures %v ( != %v)", len(weatherDataResults), requiredCompletedFutures)
	}
}

// TestGetWeatherDataFilter tests if
// the filter logic works as expected
func TestGetWeatherDataFilter(t *testing.T) {
	port := 51003
	shutdownChan := make(chan struct{})
	go startMockRPCServer(shutdownChan, port, defaultTemperatureHandler)
	defer close(shutdownChan)

	client, err := getTestRPCClient(port)
	if err != nil {
		t.Fatalf("Failed to connect to test RPC server: %v", err)
	}
	defer client.Close()

	results := make([]*Future, 0)
	idOffset := 2
	nPeers := 10 // 10
	// Expected Results: 20, 30, 40, 50, 60, ..., 110
	requiredCompletedFutures := nPeers - 2 // 8

	for i := 0; i < nPeers; i++ {
		// to start from id = idOffset
		results = append(results, GetWeatherData(client, 5*(i+idOffset)))
	}

	weatherDataResults := Wait(results, requiredCompletedFutures, 2*time.Second, heatWaveWarning)

	if len(weatherDataResults) == 0 {
		t.Errorf("Wait() returned an empty slice %v", weatherDataResults)
		return
	}

	if len(weatherDataResults) != requiredCompletedFutures {
		t.Errorf("Completed Futures %v ( != %v)", len(weatherDataResults), requiredCompletedFutures)
	}

	for i := 0; i < len(weatherDataResults); i++ {
		temp := ((weatherDataResults[i]).(WeatherDataResult).Value).(float64)
		err := (weatherDataResults[i]).(WeatherDataResult).Err
		if temp <= 35 || err != nil {
			t.Errorf("heatWaveWarning() did not filter out %v \n", temp)
			return
		}
	}
}
