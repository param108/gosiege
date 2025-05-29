package siege

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type SiegeUrl struct {
	// Url is the URL to be sieged contains
	// endpoint and query parameters
	Url     string            `json:"url"`
	Method  string            `json:"method"`  // HTTP method (GET, POST, etc.)
	Headers map[string]string `json:"headers"` // HTTP headers
	Body    string            `json:"body"`    // Request body for POST/PUT requests
	Repeat  int               `json:"repeat"`  // Number of times to repeat the request
}

type SiegeConfig struct {
	Urls          []*SiegeUrl `json:"urls"`           // List of URLs to be sieged
	Duration      int         `json:"duration"`       // Duration of the siege in seconds
	MaxConcurrent int         `json:"max_concurrent"` // Maximum number of concurrent requests
	MaxRPS        int         `json:"max_rps"`        // Maximum requests per second
}

// Siege is a struct that represents a siege
// It contains the following fields:
// Config: the configuration object for the siege
// MaxConcurrent: the maximum number of concurrent requests
// MaxRPS: the maximum requests per second
type Siege struct {
	ctx               context.Context    // Context for managing the siege lifecycle
	cancelFunc        context.CancelFunc // Function to cancel the siege context
	Config            *SiegeConfig
	ExpandedUrls      []*SiegeUrl // Expanded list of URLs with repeat counts
	counterMtx        sync.Mutex  // Mutex for synchronizing access to counters
	CurrentRPS        float64
	CurrentRequests   float64 // Current number of requests made
	CurrentConcurrent int
	MaxConcurrent     int               // Maximum number of concurrent requests allowed
	clientPool        chan *http.Client // Pool of HTTP clients for concurrent requests
	startTime         time.Time         // Start time of the siege
	waitGroup         sync.WaitGroup    // Wait group to manage goroutines
	MaxRPS            float64           // Maximum requests per second
	respMtx           sync.Mutex        // Mutex for synchronizing access to response counters
	connFailed        int               // Number of connection failures
	resp2xx           int               // Number of 200 OK responses
	resp5xx           int               // Number of 500 Internal Server Error responses
	resp4xx           int               // Number of 400 Bad Request responses
}

func (s *Siege) incrementCurrentRequests() {
	s.CurrentRequests++
	s.CurrentRPS = s.CurrentRequests / time.Since(s.startTime).Seconds()
	if s.CurrentRPS > s.MaxRPS {
		s.MaxRPS = s.CurrentRPS
	}
	s.CurrentConcurrent++
	if s.CurrentConcurrent > s.MaxConcurrent {
		s.MaxConcurrent = s.CurrentConcurrent
	}
}

func (s *Siege) decrementCurrentConcurrent() {
	s.CurrentConcurrent--
}

func NewSiege(ctx context.Context, cancel context.CancelFunc, config *SiegeConfig) *Siege {
	pool := make(chan *http.Client, config.MaxConcurrent)
	for range config.MaxConcurrent {
		pool <- &http.Client{
			Timeout: 10 * time.Second, // Set a default timeout for each client
			// You can add more client configurations here if needed
		}
	}

	siege := &Siege{
		ctx:               ctx,
		cancelFunc:        cancel,
		Config:            config,
		CurrentRequests:   0,
		CurrentConcurrent: 0,
		clientPool:        pool,
		waitGroup:         sync.WaitGroup{},
	}

	siege.expandUrlList()

	return siege
}

// expandUrlList expands the URL list in the SiegeConfig
// by repeating each URL according to its Repeat field.
func (s *Siege) expandUrlList() []*SiegeUrl {
	var expandedUrls []*SiegeUrl
	for _, url := range s.Config.Urls {
		for range url.Repeat {
			expandedUrls = append(expandedUrls, &SiegeUrl{
				Url:     url.Url,
				Method:  url.Method,
				Headers: url.Headers,
				Body:    url.Body,
				Repeat:  1, // Each expanded URL is repeated once
			})
		}
	}

	// now randomize the order of the URLs
	rand.Shuffle(len(expandedUrls), func(i, j int) {
		expandedUrls[i], expandedUrls[j] = expandedUrls[j], expandedUrls[i]
	})

	s.ExpandedUrls = expandedUrls
	return expandedUrls
}

func (s *Siege) getClient() *http.Client {
	// Get a client from the pool
	select {
	case client := <-s.clientPool:
		return client
	case <-s.ctx.Done():
		return nil // Return nil if the context is done
	}
}

func (s *Siege) returnClient(client *http.Client) {
	// Return the client to the pool
	s.clientPool <- client
}

func (s *Siege) makeCall(url *SiegeUrl, stats chan int) (*http.Response, error) {
	// Get a client from the pool
	client := s.getClient()
	if client == nil {
		return nil, context.Canceled // Return an error if the context is done
	}
	defer s.returnClient(client)

	stats <- INC // Increment the current requests count
	defer func() {
		stats <- DEC // Decrement the current concurrent requests count
	}()

	req, err := http.NewRequest(url.Method, url.Url, nil)
	if err != nil {
		return nil, err
	}

	// Set headers if any
	for key, value := range url.Headers {
		req.Header.Set(key, value)
	}

	// If the method is POST or PUT, set the body
	if url.Method == http.MethodPost || url.Method == http.MethodPut {
		req.Body = http.NoBody // Replace with actual body if needed
	}

	return client.Do(req)
}

func (s *Siege) Run(stats chan int) {
	defer s.waitGroup.Done()
	for {
		select {
		case <-s.ctx.Done():
			return // Exit if the context is done
		default:
			// Process each URL in the expanded list
			for _, url := range s.ExpandedUrls {
				resp, err := s.makeCall(url, stats)
				if err != nil {
					select {
					case <-s.ctx.Done():
						return
					default:
					}
					stats <- DISCONNECTED // Send a disconnected status to the stats channel
					// Handle error (log it, etc.)
					fmt.Printf("connection failure %v\n", err)
					continue
				}


				stats <- resp.StatusCode // Send the status code to the stats channel
				resp.Body.Close() // Ensure response body is closed
				// Process the response (e.g., log status code, etc.)
				// For now, just print the status code
			}
		}
	}
}

// GetStats returns the current statistics of the siege
// as 3 floating point numbers:
// MaxRPS, CurrentRequests, and MaxConcurrent,
// resp2xx, resp4xx, resp5xx, connFailed
func (s *Siege) GetStats() (float64, float64, int, int, int, int, int) {
	s.counterMtx.Lock()
	defer s.counterMtx.Unlock()
	return s.MaxRPS, s.CurrentRequests, s.MaxConcurrent,
		s.resp2xx, s.resp4xx, s.resp5xx, s.connFailed
}

const (
	INC          = 1
	DEC          = 2
	DISCONNECTED = 3 // Status code for disconnected or failed requests
)

// Start starts go-routines until we reach the MaxRPS limit
func (s *Siege) Start(ctxStr string) {
	s.startTime = time.Now()
	done := make(chan struct{})

	counterChan := make(chan int, 500)
	go func() {
		for {
			select {
			case v, ok := <-counterChan:
				if !ok {
					close(done) // Close the done channel when the counter channel is closed
					return
				}
				func() {
					s.counterMtx.Lock()
					defer s.counterMtx.Unlock()

					switch v {
					case INC:
						s.incrementCurrentRequests()
					case DEC:
						s.decrementCurrentConcurrent()
					case DISCONNECTED:
						s.connFailed++ // Increment connection failure count
					default:
						if v >= 200 && v < 300 {
							s.resp2xx++ // Increment 2xx response count
						} else if v >= 400 && v < 500 {
							s.resp4xx++ // Increment 4xx response count
						} else if v >= 500 {
							s.resp5xx++ // Increment 5xx response count
						}
					}
				}()
			}
		}
	}()

	// Start a goroutine to check periodically if we have reached the MaxRPS limit
	s.waitGroup.Add(1)

	// This goroutine will spawn new goroutines to run the siege
	go func() {
		defer s.waitGroup.Done()

		increaseTicker := time.NewTicker(1 * time.Second)
		defer increaseTicker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return // Exit if the context is done
			case <-increaseTicker.C:
				currentRPS, currentRequests, currentConcurrents := func() (float64, float64, int) {
					s.counterMtx.Lock()
					defer s.counterMtx.Unlock()
					return s.CurrentRPS, s.CurrentRequests, s.CurrentConcurrent
				}()
				timeOver := false

				if time.Since(s.startTime) > time.Duration(s.Config.Duration)*time.Second {
					timeOver = true
				}

				if !timeOver {
					if currentRPS < float64(s.Config.MaxRPS) && currentConcurrents < s.Config.MaxConcurrent {
						for range 10 {
							s.waitGroup.Add(1)
							go s.Run(counterChan)
						}
					}
				}

				fmt.Printf("%s current RPS: %.2f, Total Requests: %.0f, current Concurrent: %d\n",
					ctxStr, currentRPS, currentRequests, currentConcurrents)
				if timeOver {
					fmt.Println("Siege duration exceeded, stopping...")
					s.cancelFunc() // Cancel the context to stop all goroutines
					return
				}
			}
		}
	}()

	//Wait for all goroutines to finish
	s.waitGroup.Wait()
	close(counterChan) // Close the channel to signal no more increments
	<-done
}
