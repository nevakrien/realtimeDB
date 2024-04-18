package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	//	"os"
	"sort"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/montanaflynn/stats"
)

var ctx = context.Background()

func createRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
}

func main() {
	var numPublishers, numSubscribers, numMessages int
	var debug bool
	flag.IntVar(&numPublishers, "publishers", 10, "Number of publishers")
	flag.IntVar(&numSubscribers, "subscribers", 100, "Number of subscribers")
	flag.IntVar(&numMessages, "messages", 1000, "Number of messages per publisher")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.Parse()

	subscribeChannel := "messages"
	var wgSubsStart sync.WaitGroup
	//var wgPubsStart sync.WaitGroup
	var wgSubsEnd sync.WaitGroup
	var wgPubsEnd sync.WaitGroup
	allLatencies := make([][]float64, numSubscribers)

	//wait groups setup
	wgPubsEnd.Add(numPublishers)
	wgSubsStart.Add(numSubscribers)
	wgSubsEnd.Add(numSubscribers)


	// Start publishers	
	for i := 0; i < numPublishers; i++ {
		go func(publisherID int) {
			
			defer wgPubsEnd.Done()
			defer fmt.Printf("Publisher %d done\n",publisherID);

			client := createRedisClient()

			wgSubsStart.Wait()

			for j := 0; j < numMessages; j++ {
				message := fmt.Sprintf("Message %d sent at %s", j, time.Now().Format(time.RFC3339Nano))
				if debug {
					fmt.Printf("Publisher %d sending message %d\n", publisherID, j)
				}
				err := client.Publish(ctx, subscribeChannel, message).Err()
				if err != nil {
					log.Fatal(err)
				}
				time.Sleep(10 * time.Millisecond) // Throttle messages
			}
		}(i)
	}

	// Start subscribers
	for i := 0; i < numSubscribers; i++ {
		go func(subscriberID int) {
			defer wgSubsEnd.Done()
			defer fmt.Printf("Subscriber %d done\n",subscriberID);

			client := createRedisClient()
			pubsub := client.Subscribe(ctx, subscribeChannel)
			defer pubsub.Close()

			ch := pubsub.Channel()
			localLatencies := make([]float64, 0, numPublishers*numMessages)

			wgSubsStart.Done()
			for msg := range ch {
			    receivedTime := time.Now()
			    var sentTimeStr string
			    _, err := fmt.Sscanf(msg.Payload, "Message %d sent at %s", new(int), &sentTimeStr)
			    if err != nil {
			        log.Printf("Subscriber %d failed to parse message: %v", subscriberID, err)
			        continue
			    }
			    sentTime, err := time.Parse(time.RFC3339Nano, sentTimeStr)
			    if err != nil {
			        log.Printf("Subscriber %d failed to parse timestamp: %v", subscriberID, err)
			        continue
			    }
			    latency := float64(receivedTime.Sub(sentTime).Nanoseconds()) // Keep as nanoseconds
			    localLatencies = append(localLatencies, latency)
			    if len(localLatencies) == numPublishers*numMessages {
			        break // Stop after receiving the expected number of messages
			    }
			}
			allLatencies[subscriberID] = localLatencies
		}(i)
	}
	
	// Wait for all publishers and subscribers to finish
	wgSubsEnd.Wait() 
	wgPubsEnd.Wait()

	// Aggregate and sort latencies
	overallLatencies := make([]float64, 0, numSubscribers*numPublishers*numMessages)
	for _, latencies := range allLatencies {
	    overallLatencies = append(overallLatencies, latencies...)
	}
	sort.Float64s(overallLatencies)

	// Compute statistics in nanoseconds and convert to milliseconds for display
	mean, _ := stats.Mean(overallLatencies)
	percentile90, _ := stats.Percentile(overallLatencies, 90)
	percentile95, _ := stats.Percentile(overallLatencies, 95)
	percentile99, _ := stats.Percentile(overallLatencies, 99)
	percentile999, _ := stats.Percentile(overallLatencies, 99.9)

	// Display in milliseconds
	fmt.Printf("Average Latency: %.4f ms\n", mean / 1e6)
	fmt.Printf("90th Percentile Latency: %.4f ms\n", percentile90 / 1e6)
	fmt.Printf("95th Percentile Latency: %.4f ms\n", percentile95 / 1e6)
	fmt.Printf("99th Percentile Latency: %.4f ms\n", percentile99 / 1e6)
	fmt.Printf("99.9th Percentile Latency: %.4f ms\n", percentile999 / 1e6)
}
