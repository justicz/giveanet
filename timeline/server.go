package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhook"
	"github.com/unrolled/secure"

	"github.com/justicz/giveanet/common"
)

const address = "0.0.0.0:3029"
const maxWebhookLenBytes = 1 << 24
const pingInterval = 20 * time.Second
const dbPollInterval = 2 * time.Second
const durationPerMessage = 1 * time.Second

var dbInitStatements []string = []string{
	`CREATE TABLE IF NOT EXISTS messages(
		message_id serial PRIMARY KEY,
		publictoken TEXT UNIQUE,
		referral INTEGER,
		netpoints INTEGER NOT NULL,
		timeline INTEGER UNIQUE,
		message TEXT,
		displayname TEXT,
		socialtype TEXT NOT NULL,
		sociallink TEXT,
		numnets INTEGER NOT NULL,
		imgkind TEXT NOT NULL,
		imgdata BYTEA,
		created TIMESTAMPTZ,
		paidtime TIMESTAMPTZ,
		stripesessiontoken TEXT UNIQUE,
		played BOOLEAN NOT NULL,
		paid BOOLEAN NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_messages_stripesessiontoken ON messages(stripesessiontoken);`,
	`CREATE INDEX IF NOT EXISTS idx_messages_publictoken ON messages(publictoken);`,
	`CREATE INDEX IF NOT EXISTS idx_messages_timeline ON messages(timeline);`,
	`CREATE SEQUENCE IF NOT EXISTS timeline_seq;`,
}

type Timeline struct {
	mux                  sync.Mutex
	redisClient          *redis.Client
	pgClient             *sqlx.DB
	appOrigin            string
	development          bool
	stripeEndpointSecret string
	messageQueued        chan struct{}
}

// advanceTimelineForever plays through new messages one at a time from the
// database, so that if we receive e.g. 10 messages in one second, we broadcast
// them to clients slowly over time instead of all at once.
func (tl *Timeline) advanceTimelineForever() {
	for {
		// Fetch the next message in the timeline
		var messageId uint64
		var nextRowQuery = `SELECT message_id FROM messages WHERE ` +
			`paid='t' AND played='f' ORDER BY timeline ASC LIMIT 1`
		err := tl.pgClient.QueryRow(nextRowQuery).Scan(&messageId)

		// If there's no message, wait for a payment alert or the polling interval
		if err == sql.ErrNoRows {
			select {
			case <-time.After(dbPollInterval):
			case <-tl.messageQueued:
			}
			continue
		} else if err != nil {
			// Wait and try again in the hope that this was a transient issue
			log.Printf("advanceTimelineForever: unexpected sql error fetching next row: %v", err)
			time.Sleep(dbPollInterval)
			continue
		}

		log.Printf("advanceTimelineForever: started + marked message %v", messageId)

		// Mark message as played. If somehow a client sees both the new initial
		// queue and this broadcast, they won't see a dup message because they will
		// notice that they have already passed the timeline idx of the message
		_, err = tl.pgClient.Exec(`UPDATE messages SET played='t' WHERE `+
			`message_id=$1`, messageId)
		if err != nil {
			log.Printf("advanceTimelineForever: unexpected sql error marking message %v played: %v",
				messageId, err)
			time.Sleep(dbPollInterval)
			continue
		}

		// Broadcast new entry to clients and cache initial queue
		err = tl.broadcastAndCache(common.QueueResponseType, messageId, false)
		if err != nil {
			log.Printf("advanceTimelineForever: error broadcasting to redis, looping: %v", err)
			time.Sleep(dbPollInterval)
			continue
		}

		// Wait for the duration
		time.Sleep(durationPerMessage)

		log.Printf("advanceTimelineForever: finished message %v", messageId)
	}
}

func (tl *Timeline) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (tl *Timeline) stripeWebhookHandler(w http.ResponseWriter, r *http.Request) {
	// We force webhooks coming from Stripe to be single-threaded so that our
	// broadcasts of new timeline counts and messages are never out-of-order
	tl.mux.Lock()
	defer tl.mux.Unlock()

	var stripeSessionToken string
	if tl.development {
		// When developing, ignore signatures and just grab the token param
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("stripeWebhookHandler(dev): error parsing form %v", err)
			return
		}
		stripeSessionToken = r.PostForm.Get("token")
		if stripeSessionToken == "" {
			log.Printf("stripeWebhookHandler(dev): got empty token")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	} else {
		// Read request body
		limitReader := io.LimitReader(r.Body, maxWebhookLenBytes)
		body, err := ioutil.ReadAll(limitReader)
		if err != nil {
			log.Printf("stripeWebhookHandler: error reading bytes: %v", err)
			return
		}

		// In production, validate signature and parse webhook
		event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"),
			tl.stripeEndpointSecret)
		if err != nil {
			log.Printf("stripeWebhookHandler: error verifying signature: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Ignore all events that aren't payment completed events
		if event.Type != "checkout.session.completed" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Unmarshal the session
		var session stripe.CheckoutSession
		err = json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			log.Printf("stripeWebhookHandler: error parsing webhook json: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		stripeSessionToken = session.ID
	}

	// Mark this message as paid
	updatePaidQuery := `UPDATE messages SET paid='t', ` +
		`timeline=nextval('timeline_seq'), paidtime=CURRENT_TIMESTAMP WHERE ` +
		`paid='f' AND stripesessiontoken=$1 RETURNING message_id, numnets`
	var messageId uint64
	var numNets uint64
	row := tl.pgClient.QueryRow(updatePaidQuery, stripeSessionToken)
	err := row.Scan(&messageId, &numNets)
	if err == sql.ErrNoRows {
		// Don't log stripe token
		log.Printf("stripeWebhookHandler: row does not exist or was already paid for")
		w.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("stripeWebhookHandler: unexpected database error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Past this point we'll always return OK since we marked the message as paid
	w.WriteHeader(http.StatusOK)

	// Alert the timeline goroutine that it should check the database again since
	// a new payment just came in. Channel is buffered with capacity 1 so if we
	// receive a webhook right after checking the database but before the
	// timeline goroutine starts waiting on it, we won't miss it.
	select {
	case tl.messageQueued <- struct{}{}:
	default:
	}

	// Update the leaderboard recursively, following the referral parameter
	const leaderboardUpdateQuery = `
	WITH RECURSIVE ancestors(mid) AS (
			SELECT CAST($1 AS INTEGER)
		UNION ALL
			SELECT m.referral FROM messages m, ancestors a WHERE m.message_id = a.mid
	)
	UPDATE messages m SET netpoints = netpoints + $2 FROM ancestors a WHERE m.message_id = a.mid`
	_, err = tl.pgClient.Exec(leaderboardUpdateQuery, messageId, numNets)
	if err != nil {
		log.Printf("stripeWebhookHandler: unexpected sql error updating leaderboard: %v", err)
		// Fallthrough
	}

	// Invalidate page caches
	err = tl.invalidateCachedPages()
	if err != nil {
		log.Printf("stripeWebhookHandler: failed to invalidate page cache: %v", err)
		// Fallthrough
	}

	// Broadcast and cache new net count
	err = tl.broadcastAndCache(common.NetsGivenResponseType, messageId, false)
	if err != nil {
		log.Printf("stripeWebhookHandler: failed to broadcastAndCache [0]: %v", err)
		// Fallthrough
	}

	// Broadcast and cache new leaderboard
	err = tl.broadcastAndCache(common.LeaderboardResponseType, messageId, false)
	if err != nil {
		log.Printf("stripeWebhookHandler: failed to broadcastAndCache [1]: %v", err)
		// Fallthrough
	}

	// The broadcast and caching for the message queue takes place in
	// advanceTimelineForever, since we want to space out message broadcasts

	return
}

func (tl *Timeline) updateInitialGoalCache(resp common.Response) (err error) {
	// Pull out how many nets have been given
	if resp.NetsGivenResponse == nil {
		return fmt.Errorf("NetsGivenResponse was nil")
	}
	netsGiven := resp.NetsGivenResponse.NetsGiven

	// Select the appropriate goal
	var goal common.Goal
	for _, newGoal := range common.Goals {
		goal = newGoal
		if netsGiven < goal.NumNets {
			break
		}
	}

	err = tl.redisClient.MSet(common.InitialGoalKey, goal.Name,
		common.InitialNinesKey, goal.Nines).Err()
	if err != nil {
		return fmt.Errorf("failed to update initial goal keys %v", err)
	}

	return
}

func (tl *Timeline) invalidateCachedPages() (err error) {
	pageFmts := []string{common.LeaderboardCachePageFmt, common.QueueCachePageFmt}
	keysToDelete := make([]string, 0, len(pageFmts)*common.PagesToCache)
	for _, pageFmt := range pageFmts {
		for i := 0; i < common.PagesToCache; i++ {
			keysToDelete = append(keysToDelete, fmt.Sprintf(pageFmt, i))
		}
	}
	err = tl.redisClient.Del(keysToDelete...).Err()
	if err != nil {
		return fmt.Errorf("failed to invalidate page cache %v", err)
	}
	return
}

func (tl *Timeline) broadcastAndCache(responseType string, messageId uint64, startup bool) (err error) {
	// Make a response of the passed type by querying the database
	var cacheResp, bcastResp common.Response
	var updateChannel, cacheKey string
	switch responseType {
	case common.NetsGivenResponseType:
		updateChannel = common.NetsGivenUpdateChannel
		cacheKey = common.InitialNetsGivenKey
		cacheResp, bcastResp, err = tl.getNetsGivenUpdate()
		// When recomputing the number of nets given, update the initial values
		// for the counter on the homepage
		err = tl.updateInitialGoalCache(cacheResp)
		if err != nil {
			return
		}
	case common.QueueResponseType:
		updateChannel = common.QueueUpdateChannel
		cacheKey = common.InitialQueueKey
		cacheResp, bcastResp, err = tl.getQueueUpdate(messageId, startup)
	case common.LeaderboardResponseType:
		updateChannel = common.LeaderboardUpdateChannel
		cacheKey = common.InitialLeaderboardKey
		cacheResp, bcastResp, err = tl.getLeaderboardUpdate()
	default:
		log.Fatalf("broadcastAndCache: unexpected responseType argument: %v", responseType)
	}
	if err != nil {
		return
	}

	// Serialize response to cache to json
	cacheRespSerialized, err := json.Marshal(cacheResp)
	if err != nil {
		log.Printf("broadcastAndCache: error serializing cacheResp %v: %v", responseType, err)
		return
	}

	// Serialize response to broadcast to json
	bcastRespSerialized, err := json.Marshal(bcastResp)
	if err != nil {
		log.Printf("broadcastAndCache: error serializing bcastResp %v: %v", responseType, err)
		return
	}

	// Update cache
	err = tl.redisClient.Set(cacheKey, cacheRespSerialized, 0).Err()
	if err != nil {
		log.Printf("broadcastAndCache: failed to update cache for %v: %v", responseType, err)
		return
	}

	// We don't need to broadcast anything on startup
	if !startup {
		// Publish to the redis channel so that we can send updated net
		// counts to clients
		_, err = tl.redisClient.Publish(updateChannel, bcastRespSerialized).Result()
		if err != nil {
			log.Printf("broadcastAndCache: failed to redis broadcast for %v: %v", responseType, err)
			return
		}
	}

	return
}

func (tl *Timeline) getNetsGivenUpdate() (cacheResp, bcastResp common.Response, err error) {
	// Cache & broadcast the number of nets given
	resp, err := common.GetNetsGivenResponse(tl.pgClient)
	if err != nil {
		return
	}
	return resp, resp, nil
}

func (tl *Timeline) getQueueUpdate(messageId uint64, startup bool) (cacheResp, bcastResp common.Response, err error) {
	// Cache the 0th page of messages
	cacheResp, err = common.GetQueueResponse(tl.pgClient, tl.appOrigin, 0)
	if err != nil {
		return
	}

	if !startup {
		// Fetch the new message to be broadcast
		var bcastDBEntry common.DBQueueEntry
		const messageQuery = `SELECT * from messages WHERE message_id=$1`
		err = tl.pgClient.Get(&bcastDBEntry, messageQuery, messageId)
		if err != nil {
			log.Printf("getQueueUpdate: failed to get message to broadcast: %v", err)
			return
		}

		// Fill in the response to be broadcast
		bcastEntry := common.DBQueueEntryToQueueEntry(tl.appOrigin, bcastDBEntry)
		bcastResp = common.Response{
			Type: common.QueueResponseType,
			QueueResponse: &(common.QueueResponse{
				Entries: []common.QueueEntry{bcastEntry},
			}),
		}
	}

	return
}

func (tl *Timeline) getLeaderboardUpdate() (cacheResp, bcastResp common.Response, err error) {
	// Cache & broadcast the 0th page of the leaderboard
	resp, err := common.GetLeaderboardResponse(tl.pgClient, tl.appOrigin, 0)
	if err != nil {
		return
	}
	return resp, resp, nil
}

func initializeDatabase(dbClient *sqlx.DB) error {
	for _, stmt := range dbInitStatements {
		_, err := dbClient.Exec(stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (tl *Timeline) publishPingsForever() {
	// Marshal ping response, which never changes
	pingBytes, err := json.Marshal(common.Response{Type: common.PingResponseType})
	if err != nil {
		log.Fatalf("publishPingsForever: failed to marshal ping: %v", err)
	}

	// []byte -> string
	pingResponse := string(pingBytes)

	// Broadcast ping every pingInterval
	tick := time.Tick(pingInterval)
	for {
		select {
		case <-tick:
			err := tl.redisClient.Publish(common.PingChannel, pingResponse).Err()
			if err != nil {
				log.Printf("publishPingsForever: failed to publish ping to redis channel: %v", err)
			}
		}
	}
}

func main() {
	// Ensure required environment variables are set
	webhookSecret := os.Getenv("MN_WEBHOOK_SECRET")
	if webhookSecret == "" {
		log.Fatalf("MN_WEBHOOK_SECRET (webhook url secret prefix) environment variable is required")
	}

	stripeSecret := os.Getenv("MN_STRIPE_SECRET")
	if stripeSecret == "" {
		log.Fatalf("MN_STRIPE_SECRET (stripe secret token) environment variable is required")
	}

	stripeEndpointSecret := os.Getenv("MN_STRIPE_ENDPOINT_SECRET")
	if stripeEndpointSecret == "" {
		log.Fatalf("MN_STRIPE_ENDPOINT_SECRET (stripe endpoint secret) environment variable is required")
	}

	appOrigin := os.Getenv("MN_APP_ORIGIN")
	if appOrigin == "" {
		log.Fatalf("MN_APP_ORIGIN (application server origin) environment variable is required")
	}

	postgresURL := os.Getenv("MN_POSTGRES_URL")
	if postgresURL == "" {
		log.Fatalf("MN_POSTGRES_URL (postgres connection URL) environment variable is required")
	}

	environment := os.Getenv("MN_ENVIRONMENT")
	if environment == "" {
		log.Fatalf("MN_ENVIRONMENT (deploy environment) environment variable is required")
	}

	redisAddr := os.Getenv("MN_REDIS_ADDR")
	if redisAddr == "" {
		log.Fatalf("MN_REDIS_ADDR (redis address) environment variable is required")
	}

	redisPassword := os.Getenv("MN_REDIS_PASS")

	// Initialize stripe library
	stripe.Key = stripeSecret

	// Build redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})

	// Check redis connection works
	var i int
	const numTries = 10
	for i = 0; i < numTries; i++ {
		pong, err := redisClient.Ping().Result()
		if err != nil {
			log.Printf("failed to connect to redis (try %d/%d): %v", i+1, numTries, err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("connected to redis: %v response: %v", redisAddr, pong)
		break
	}
	if i == numTries {
		log.Fatalf("failed to connect to redis")
	}

	// Build postgres client
	pgClient, err := sqlx.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("failed to create postgres client: %v", err)
	}

	// Check postgres connection works
	for i = 0; i < numTries; i++ {
		err = pgClient.Ping()
		if err != nil {
			log.Printf("failed to connect to postgres (try %d/%d): %v", i+1, numTries, err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("connected to postgres")
		break
	}
	if i == numTries {
		log.Fatalf("failed to connect to postgres")
	}

	// Are we a development environment?
	development := strings.HasPrefix(strings.ToLower(environment), "dev")

	// Construct timeline service
	timeline := Timeline{
		redisClient:          redisClient,
		pgClient:             pgClient,
		appOrigin:            appOrigin,
		development:          development,
		stripeEndpointSecret: stripeEndpointSecret,
		messageQueued:        make(chan struct{}, 1),
	}

	// Initialize database if required
	err = initializeDatabase(pgClient)
	if err != nil {
		log.Fatalf("failed to initialize postgres database: %v", err)
	}

	// Start the message timeline
	go timeline.advanceTimelineForever()

	// Publish periodic pings
	go timeline.publishPingsForever()

	// Populate cache
	messageTypes := []string{
		common.NetsGivenResponseType,
		common.LeaderboardResponseType,
		common.QueueResponseType,
	}
	for _, t := range messageTypes {
		err = timeline.broadcastAndCache(t, 0, true)
		if err != nil {
			log.Fatalf("failed to initialize cache for %v: %v", t, err)
		}
	}

	// Invalidate any pages already in the cache
	err = timeline.invalidateCachedPages()
	if err != nil {
		log.Fatalf("failed to invalidate page cache: %v", err)
	}

	// Mark cache as ready
	err = timeline.redisClient.Set(common.StartupKey, "done", 0).Err()
	if err != nil {
		log.Fatalf("failed to write startup key to redis: %v", err)
	}

	// Start webhook handler server
	r := mux.NewRouter()
	r.HandleFunc(fmt.Sprintf("/api/webhook/%s/stripe", webhookSecret),
		timeline.stripeWebhookHandler).Methods("POST")
	r.HandleFunc("/health", timeline.healthHandler).Methods("GET")

	// Set security headers
	secureMiddleware := secure.New(secure.Options{
		STSSeconds:           31536000,
		STSIncludeSubdomains: true,
		STSPreload:           true,
		FrameDeny:            true,
		ContentTypeNosniff:   true,
		SSLProxyHeaders:      map[string]string{"X-Forwarded-Proto": "https"},
		IsDevelopment:        development,
	})

	srv := &http.Server{
		Handler:      secureMiddleware.Handler(r),
		Addr:         address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Starting server on %s", address)
	srv.ListenAndServe()
}
