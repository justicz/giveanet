package main

import (
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis"

	"github.com/justicz/giveanet/common"
)

const broadcastChanCapacity = 128

type MessageNotifier struct {
	mux           sync.Mutex
	redisClient   *redis.Client
	subscribers   map[uint64](chan string)
	lastClientIdx uint64
}

func (mn *MessageNotifier) subscribeClient() (uint64, <-chan string) {
	mn.mux.Lock()
	defer mn.mux.Unlock()
	// Bump lastClientIdx to get ID for this client
	mn.lastClientIdx++
	// Create channel for client to receive messages
	ch := make(chan string, broadcastChanCapacity)
	mn.subscribers[mn.lastClientIdx] = ch
	// Return ID for client + channel itself
	return mn.lastClientIdx, ch
}

func (mn *MessageNotifier) unsubscribeClient(idx uint64) {
	mn.mux.Lock()
	defer mn.mux.Unlock()
	delete(mn.subscribers, idx)
}

func (mn *MessageNotifier) handleBroadcastMessages() {
	channels := []string{
		common.NetsGivenUpdateChannel,
		common.QueueUpdateChannel,
		common.LeaderboardUpdateChannel,
		common.PingChannel,
	}

	// Subscribe to broadcast redis channels
	pubsub := mn.redisClient.Subscribe(channels...)
	_, err := pubsub.Receive()
	if err != nil {
		log.Fatalf("MessageNotifier: failed to subscribe to redis channel: %v\n", err)
	}

	// If we exit for some reason, close pubsub
	defer pubsub.Close()
	ch := pubsub.Channel()
	var redisMsg *redis.Message
	for {
		// Grab next message from subscription
		redisMsg = <-ch
		if redisMsg == nil {
			log.Printf("MessageNotifier: redis subscription channel unexpectedly returned nil")
			time.Sleep(1 * time.Second)
			continue
		}

		// Sanity check that this came from an expected channel
		found := false
		for _, name := range channels {
			if redisMsg.Channel == name {
				found = true
				break
			}
		}
		if !found {
			log.Printf("MessageNotifier: got message on unexpected channel: %v", redisMsg.Channel)
			time.Sleep(1 * time.Second)
			continue
		}

		// Broadcast message to all subscribers (message already serialized)
		if redisMsg.Payload != "" {
			mn.mux.Lock()
			log.Printf("MessageNotifier: notifying %d subscribers from redis "+
				"channel %s", len(mn.subscribers), redisMsg.Channel)
			for _, ch := range mn.subscribers {
				select {
				case ch <- redisMsg.Payload:
				default:
				}
			}
			mn.mux.Unlock()
		}
	}
}
