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

	"github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)



var ctx = context.Background()
	
// MessageBroker defines the interface for a message broker system that can publish and subscribe to messages.
// MessageBroker defines the interface for a message broker system that can publish messages.
type MessageBroker interface {
    Subscribe(ctx context.Context, channel string) Subscription
    Publish(ctx context.Context, channel string, message string) error
}

// Subscription defines the interface for managing message subscriptions.
type Subscription interface {
    Channel() <-chan Message
    Close() error
}

// Message defines the interface for a message in a pub/sub system.
type Message interface {
    GetMessageChannel() string
    GetPayload() string
}


type RedisMessageBroker struct {
    client *redis.Client
}

type RedisSubscription struct {
    pubsub *redis.PubSub
}

type RedisMessageAdapter struct {
    msg *redis.Message
}

func NewRedisMessageAdapter(msg *redis.Message) *RedisMessageAdapter {
    return &RedisMessageAdapter{msg: msg}
}

func (rma *RedisMessageAdapter) GetMessageChannel() string {
    return rma.msg.Channel
}

func (rma *RedisMessageAdapter) GetPayload() string {
    return rma.msg.Payload
}

// Global variable to hold Redis container details
var redisContainer testcontainers.Container
var redisURI string
var redisType string

func setupDB(ctx context.Context) error {
	// Mapping from type to Docker image
	redisImages := map[string]string{
		"keydb":  "eqalpha/keydb:x86_64_v6.0.16",
		"redis":  "redis:latest",
		"valkey": "valkey/valkey:unstable", // Replace with actual image if available
	}

	image, ok := redisImages[redisType]
	if !ok {
		return fmt.Errorf("unsupported redis type: %s", redisType)
	}

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	var err error
	redisContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}

	// Getting the mapped port
	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		return err
	}
	redisURI = fmt.Sprintf("localhost:%s", port.Port())
	return nil
}

// func NewRedisMessageBroker() *redis.Client {
//     return redis.NewClient(&redis.Options{
//         Addr: redisURI,
//     })
// }
func NewRedisMessageBroker() *RedisMessageBroker {
    return &RedisMessageBroker{
        client: redis.NewClient(&redis.Options{
            Addr: redisURI,
        }),
    }
}

func (rs *RedisSubscription) Channel() <-chan Message {
    output := make(chan Message)
    go func() {
        defer close(output)
        for msg := range rs.pubsub.Channel() {
            adaptedMsg := NewRedisMessageAdapter(msg)
            output <- adaptedMsg
        }
    }()
    return output
}

func (rs *RedisSubscription) Close() error {
    return rs.pubsub.Close()
}

func (r *RedisMessageBroker) Publish(ctx context.Context, channel string, message string) error {
    return r.client.Publish(ctx, channel, message).Err()
}

func (r *RedisMessageBroker) Subscribe(ctx context.Context, channel string) Subscription {
    pubsub := r.client.Subscribe(ctx, channel)
    // Wrap the *redis.PubSub in your RedisSubscription struct
    return &RedisSubscription{pubsub: pubsub}
}


func main() {
	var numPublishers, numSubscribers, numMessages int
	var debug bool 
	//flag.BoolVar

	flag.StringVar(&redisType, "redis-type", "redis", "Type of Redis to use (keydb, redis, valkey)")

	flag.IntVar(&numPublishers, "publishers", 10, "Number of publishers")
	flag.IntVar(&numSubscribers, "subscribers", 10, "Number of subscribers")
	flag.IntVar(&numMessages, "messages", 1000, "Number of messages per publisher")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode")
	flag.Parse()

	ctx := context.Background()
    err := setupDB(ctx)
    if err != nil {
        log.Fatalf("Could not set up Redis container: %v", err)
        return;
    }
    defer redisContainer.Terminate(ctx)

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
			//defer fmt.Printf("Publisher %d done\n",publisherID);

			client := NewRedisMessageBroker()

			wgSubsStart.Wait()

			for j := 0; j < numMessages; j++ {
				time.Sleep(3 * time.Millisecond) // Throttle messages

				message := fmt.Sprintf("Message %d sent at %s", j, time.Now().Format(time.RFC3339Nano))
				if debug {
					fmt.Printf("Publisher %d sending message %d\n", publisherID, j)
				}
				err := client.Publish(ctx, subscribeChannel, message)//.Err()
				if err != nil {
					log.Fatal(err)
				}
			}
		}(i)
	}

	// Start subscribers
	for i := 0; i < numSubscribers; i++ {
		go func(subscriberID int) {
			defer wgSubsEnd.Done()
			//defer fmt.Printf("Subscriber %d done\n",subscriberID);

			client := NewRedisMessageBroker()
			pubsub := client.Subscribe(ctx, subscribeChannel)
			defer pubsub.Close()

			ch := pubsub.Channel()
			localLatencies := make([]float64, 0, numPublishers*numMessages)

			wgSubsStart.Done()
			for msg := range ch {
			    receivedTime := time.Now()
			    var sentTimeStr string
			    _, err := fmt.Sscanf(msg.GetPayload(), "Message %d sent at %s", new(int), &sentTimeStr)
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

	// Assuming overallLatencies is sorted
	maxLatency := overallLatencies[len(overallLatencies)-1] // Last element after sort
	fmt.Printf("Max Latency: %.4f ms\n", maxLatency/1e6)


	// Display in milliseconds
	fmt.Printf("Average Latency: %.4f ms\n", mean / 1e6)
	fmt.Printf("90th Percentile Latency: %.4f ms\n", percentile90 / 1e6)
	fmt.Printf("95th Percentile Latency: %.4f ms\n", percentile95 / 1e6)
	fmt.Printf("99th Percentile Latency: %.4f ms\n", percentile99 / 1e6)
	fmt.Printf("99.9th Percentile Latency: %.4f ms\n", percentile999 / 1e6)
}
