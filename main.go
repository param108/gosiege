package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"github.com/param108/siege/config"
	"github.com/param108/siege/siege"
	"syscall"

	"github.com/urfave/cli/v3"
)

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func main() {
	// use urfave/cli to build a commandline tool with name siege which takes one
	// mandatory parameter -c the path to the config file.
	cmd := &cli.Command{
		Name:  "sg",
		Usage: "A commandline tool to run siege tests",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run the siege test with the provided configuration",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "config",
						Aliases:  []string{"c"},
						Usage:    "Path to the configuration file",
						Required: true,
					},
					&cli.IntFlag{
						Name:    "max-rps",
						Aliases: []string{"r"},
						Usage:   "Maximum requests per second",
					},
					&cli.IntFlag{
						Name:    "max-concurrent",
						Aliases: []string{"m"},
						Usage:   "Maximum number of concurrent requests",
					},
					&cli.IntFlag{
						Name:    "duration",
						Aliases: []string{"d"},
						Usage:   "Duration of the siege in seconds",
						Value:   60, // Default duration is 60 seconds
					},
				},
				Action: func(clictx context.Context, c *cli.Command) error {
					configPath := c.String("config")
					// parse the config file and run the siege test
					siegeConfig, err := config.ParseConfig(configPath)
					if err != nil {
						return cli.Exit("Error parsing config file: "+err.Error(), 1)
					}

					// Override max-rps and max-concurrent if provided
					if c.Int("max-rps") > 0 {
						siegeConfig.MaxRPS = c.Int("max-rps")
					}

					if c.Int("max-concurrent") > 0 {
						siegeConfig.MaxConcurrent = c.Int("max-concurrent")
					}

					// Set the duration for the siege
					if c.Int("duration") > 0 {
						siegeConfig.Duration = c.Int("duration")
					}

					ctx, cancel := context.WithCancel(context.Background())

					// create a channel to catch signal interrupts
					signalChan := make(chan os.Signal, 1)
					signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
					go func() {
						<-signalChan
						cancel() // cancel the context on interrupt
					}()

					siege := siege.NewSiege(ctx, cancel, siegeConfig)

					//ctxStr to a random string of length 10 using only alphanumeric characters
					ctxStr := generateRandomString(10)

					// Run the siege test
					siege.Start(ctxStr)

					// get the last stats and print them
					maxRPS, currentRequests, maxConcurrents,
						resp2xx, resp4xx, resp5xx, connFailed := siege.GetStats()
					fmt.Printf(`Final Stats:
Max RPS: %.2f
Current Requests: %.2f
Max Concurrents: %d
2xx Responses: %d
4xx Responses: %d
5xx Responses: %d
connection failures: %d
`,
						maxRPS, currentRequests, maxConcurrents,
						resp2xx, resp4xx, resp5xx, connFailed)
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatalf("Error running command: %v", err)
	}
}
