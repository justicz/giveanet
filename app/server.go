package main

import (
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/checkout/session"
	"github.com/unrolled/secure"
	"golang.org/x/crypto/blowfish"
	"golang.org/x/net/idna"
	"golang.org/x/net/websocket"

	"github.com/justicz/giveanet/common"
)

const clientChanCapacity = 128
const appAddress = "0.0.0.0:3021"
const wsAddress = "0.0.0.0:3024"
const publicTokenLenBytes = 16
const internalServerError = "Internal server error"
const badRequest = "Bad request"
const referralCookie = "referral"
const tokenLength = 13
const maxPageNumber = 1000000
const maxPayloadBytes = 1 << 19

// We send a ping every 20 seconds, so we should see a response about as often
const websocketReadTimeout = 30 * time.Second
const websocketWriteTimeout = 10 * time.Second

var websocketNoTimeout = time.Time{}

type RequestContext struct {
	messageNotifier        *MessageNotifier
	pgClient               *sqlx.DB
	redisClient            *redis.Client
	template               *template.Template
	appOrigin              string
	wsOrigin               string
	stripePublicToken      string
	tokenPermutationSecret []byte
	development            bool
}

func (rctx *RequestContext) homeHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch homepage params. Ignore errors so we will always serve at least
	// default parameters
	goal, _ := rctx.fetchHomepageGoalParams()

	w.WriteHeader(http.StatusOK)
	rctx.template.ExecuteTemplate(w, "home", HomePageData{
		WSOrigin:     rctx.wsOrigin,
		InitialGoal:  goal.Name,
		InitialNines: goal.Nines,
	})
}

func (rctx *RequestContext) fetchHomepageGoalParams() (goal common.Goal, err error) {
	// Fetch the current goal from the cache
	encoded, err := rctx.redisClient.Get(common.InitialGoalKey).Result()
	if err != nil {
		log.Printf("error fetching goal key from redis: %v", err)
		return common.DefaultGoal, err
	}

	// Unmarshal the goal
	err = json.Unmarshal([]byte(encoded), &goal)
	if err != nil {
		log.Printf("error unmarshalling goal from redis: %v", err)
		return common.DefaultGoal, err
	}

	return
}

func (rctx *RequestContext) sendHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	rctx.template.ExecuteTemplate(w, "send", map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
	})
}

func (rctx *RequestContext) parsePageNumber(r *http.Request, optional bool) (page uint64, err error) {
	// Convert page number to integer & validate (if passed)
	pageParam, ok := mux.Vars(r)["page"]
	if ok {
		page, err = strconv.ParseUint(pageParam, 10, 64)
		if err != nil || page == 0 || page > maxPageNumber {
			return page, fmt.Errorf("invalid page number")
		}
		return page, nil
	} else if !optional {
		return page, fmt.Errorf("missing page number")
	}
	return page, nil
}

func (rctx *RequestContext) allHandler(w http.ResponseWriter, r *http.Request) {
	_, err := rctx.parsePageNumber(r, true)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		rctx.template.ExecuteTemplate(w, "error", ErrorPageData{http.StatusBadRequest})
		return
	}
	w.WriteHeader(http.StatusOK)
	rctx.template.ExecuteTemplate(w, "all", nil)
}

func (rctx *RequestContext) leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	_, err := rctx.parsePageNumber(r, true)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		rctx.template.ExecuteTemplate(w, "error", ErrorPageData{http.StatusBadRequest})
		return
	}
	w.WriteHeader(http.StatusOK)
	rctx.template.ExecuteTemplate(w, "leaderboard", nil)
}

// commonPageHandler handles common code for both rankingsHandler ond messagesHandler
func (rctx *RequestContext) commonPageHandler(w http.ResponseWriter, r *http.Request,
	databasePager func(*sqlx.DB, string, uint64) (common.Response, error),
	cacheFmt string) {
	w.Header().Set("Content-Type", "application/json")

	// Convert page number to integer & validate
	page, err := rctx.parsePageNumber(r, false)
	if err != nil {
		badRequestJSONResponse(w)
		return
	}

	// User pages are 1-indexed
	page = page - 1

	// Check cache for first PagesToCache
	var shouldCache bool
	cacheKey := fmt.Sprintf(cacheFmt, page)
	if page < common.PagesToCache {
		cached, err := rctx.redisClient.Get(cacheKey).Result()
		if err != nil && err != redis.Nil {
			log.Printf("unexpected error from redis when checking cache: %v", err)
			// Fallthrough
		} else if err == nil {
			// Return cached page
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, cached)
			return
		} else if err == redis.Nil {
			// Key didn't exist but should
			shouldCache = true
		}
	}

	// Pull from database
	resp, err := databasePager(rctx.pgClient, rctx.appOrigin, page)
	if err != nil {
		internalServerErrorJSONResponse(w)
		return
	}

	// Encode result
	encoded, err := json.Marshal(resp)
	if err != nil {
		log.Printf("failed to encode response: %v", err)
		internalServerErrorJSONResponse(w)
		return
	}

	// Cache the result (invalidated by timeline)
	if shouldCache {
		err = rctx.redisClient.Set(cacheKey, encoded, 0).Err()
		if err != nil {
			log.Printf("unexpected error from redis when setting cache: %v", err)
			// Fallthrough
		}
	}

	// Return response
	w.WriteHeader(http.StatusOK)
	w.Write(encoded)
	return
}

func (rctx *RequestContext) rankingsHandler(w http.ResponseWriter, r *http.Request) {
	lbType, ok := r.URL.Query()["t"]
	if ok && len(lbType) == 1 && lbType[0] == "country" {
		rctx.commonPageHandler(w, r, common.GetCountryLeaderboardResponse, common.CountryLeaderboardCachePageFmt)
		return
	}
	rctx.commonPageHandler(w, r, common.GetLeaderboardResponse, common.LeaderboardCachePageFmt)
}

func (rctx *RequestContext) messagesHandler(w http.ResponseWriter, r *http.Request) {
	rctx.commonPageHandler(w, r, common.GetQueueResponse, common.QueueCachePageFmt)
}

func (rctx *RequestContext) stripeInitHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "var stripe = Stripe('%s');", rctx.stripePublicToken)
}

func normalizeLink(link string) (normalized string, err error) {
	// Parse link as URL
	u, err := url.Parse(link)
	if err != nil {
		return
	}

	// Punycode any non-ascii hosts
	u.Host, err = idna.ToASCII(u.Host)
	if err != nil {
		return
	}

	return u.String(), nil
}

func internalServerErrorJSONResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	resp := common.PayFormResponse{
		Errors: []string{internalServerError},
	}
	json.NewEncoder(w).Encode(resp)
}

func badRequestJSONResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	resp := common.PayFormResponse{
		Errors: []string{badRequest},
	}
	json.NewEncoder(w).Encode(resp)
}

func (rctx *RequestContext) retryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var publicToken string
	var numNets uint64
	const linkQuery = `SELECT publictoken, numnets FROM messages WHERE ` +
		`publictoken=$1 AND paid='f'`
	row := rctx.pgClient.QueryRow(linkQuery, mux.Vars(r)["token"])
	err := row.Scan(&publicToken, &numNets)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		resp := common.CardResponse{
			Errors: []string{"Card not found"},
		}
		json.NewEncoder(w).Encode(resp)
		return
	} else if err != nil {
		log.Printf("retryHandler: unexpected database error fetching message: %v", err)
		internalServerErrorJSONResponse(w)
		return
	}

	// Create a new stripe session and update row
	stripeSession, err := rctx.makeStripeSession(numNets, publicToken, false)
	if err != nil {
		log.Printf("retryHandler: error creating stripe session: %v", err)
		internalServerErrorJSONResponse(w)
		return
	}

	// Update stripe session token in database
	const updateQuery = `UPDATE messages SET stripesessiontoken=$1 WHERE ` +
		`publictoken=$2 AND paid='f'`
	_, err = rctx.pgClient.Exec(updateQuery, stripeSession, publicToken)
	if err != nil {
		log.Printf("retryHandler: error updating stripe session: %v", err)
		internalServerErrorJSONResponse(w)
	}

	// Success! Redirect user to stripe
	resp := common.PayFormResponse{
		StripeSession: stripeSession,
	}
	json.NewEncoder(w).Encode(resp)
	return
}

func (rctx *RequestContext) makeStripeSession(numNets uint64, publicToken string, private bool) (string, error) {
	// If development, don't hit stripe
	if rctx.development {
		return makeFakeStripeSession()
	}

	// Succesful payment page for message
	successURL := fmt.Sprintf("%s/card/%s?r=paid", rctx.appOrigin, publicToken)
	if private {
		successURL = fmt.Sprintf("%s/thankyou", rctx.appOrigin)
	}

	// Canceled payment (retry) page for message
	cancelURL := fmt.Sprintf("%s/card/%s?r=cancel", rctx.appOrigin, publicToken)
	if private {
		cancelURL = fmt.Sprintf("%s/", rctx.appOrigin)
	}

	params := &stripe.CheckoutSessionParams{
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			&stripe.CheckoutSessionLineItemParams{
				Name:        stripe.String("Mosquito Net"),
				Description: stripe.String("Long-Lasting Insecticidal Net"),
				Amount:      stripe.Int64(common.NetCostCents),
				Currency:    stripe.String(string(stripe.CurrencyUSD)),
				Quantity:    stripe.Int64(int64(numNets)),
			},
		},
		SuccessURL: stripe.String(successURL),
		CancelURL:  stripe.String(cancelURL),
	}

	// Make stripe session and return session ID
	stripeSession, err := session.New(params)
	return stripeSession.ID, err
}

func (rctx *RequestContext) payHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse and validate form data, which can generate multiple errors
	formData, errors := validateNewMessageForm(r)
	if len(errors) > 0 {
		// If there are errors, serialize them and respond
		w.WriteHeader(http.StatusBadRequest)
		resp := common.PayFormResponse{
			Errors: errors,
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Check for a referral code
	var referralCode string
	for _, cookie := range r.Cookies() {
		if cookie.Name == referralCookie {
			// Basic validation
			if len(cookie.Value) == tokenLength {
				referralCode = cookie.Value
			}
		}
	}

	// Look up the message_id for referral code, don't fail if it's missing
	var referralId *uint64
	if referralCode != "" {
		const findReferralQuery = `SELECT message_id FROM messages WHERE publictoken=$1`
		err := rctx.pgClient.Get(&referralId, findReferralQuery, referralCode)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("payHandler: error looking up referral code: %v", err)
			internalServerErrorJSONResponse(w)
			return
		}
	}

	/*
	 * Add to database and get ID
	 */

	fd := formData
	insertQuery := `INSERT INTO messages (message, displayname, numnets, ` +
		`imgkind, imgdata, socialtype, sociallink, country, referral, paid, ` +
		`played, netpoints, created) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, ` +
		`'f', 'f', 0, CURRENT_TIMESTAMP) RETURNING message_id`
	row := rctx.pgClient.QueryRow(insertQuery, fd.Message, fd.DisplayName,
		fd.NumNets, fd.ImgKind, fd.ImgData, fd.SocialLinkType, fd.SocialLink,
		fd.Country, referralId)

	var messageId uint64
	err := row.Scan(&messageId)
	if err != nil {
		log.Printf("payHandler: error during insert: %v", err)
		internalServerErrorJSONResponse(w)
		return
	}

	/*
	 * Generate unique and short public ID using a short block cipher
	 */
	cipher, err := blowfish.NewCipher(rctx.tokenPermutationSecret)
	if err != nil {
		log.Printf("payHandler: couldn't generate public token: %v", err)
		internalServerErrorJSONResponse(w)
		return
	}

	var intBytes [8]byte
	binary.LittleEndian.PutUint64(intBytes[:], messageId)

	var publicTokenBytes [8]byte
	cipher.Encrypt(publicTokenBytes[:], intBytes[:])

	var publicToken string
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	publicToken = encoder.EncodeToString(publicTokenBytes[:])

	/*
	 * Create stripe session
	 */
	stripeSession, err := rctx.makeStripeSession(fd.NumNets, publicToken, fd.Private)
	if err != nil {
		log.Printf("payHandler: error creating stripe session: %v", err)
		internalServerErrorJSONResponse(w)
		return
	}

	/*
	 * Update entry with public token and stripe token
	 */
	updateTokensQuery := `UPDATE messages SET stripesessiontoken=$1, ` +
		`publictoken=$2 WHERE message_id=$3`
	_, err = rctx.pgClient.Exec(updateTokensQuery, stripeSession, publicToken, messageId)
	if err != nil {
		log.Printf("payHandler: error adding tokens to database: %v", err)
		internalServerErrorJSONResponse(w)
		return
	}

	// Success! Reply with stripe token so client can redirect
	w.WriteHeader(http.StatusOK)
	resp := common.PayFormResponse{
		StripeSession: stripeSession,
	}
	json.NewEncoder(w).Encode(resp)
	return
}

func (rctx *RequestContext) cardHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure card exists
	var publicToken string
	var numNets uint64
	var paid bool
	const checkMessageQuery = `SELECT publictoken, paid, numnets FROM messages ` +
		`WHERE publictoken=$1`
	row := rctx.pgClient.QueryRow(checkMessageQuery, mux.Vars(r)["token"])
	err := row.Scan(&publicToken, &paid, &numNets)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		rctx.template.ExecuteTemplate(w, "error", ErrorPageData{http.StatusNotFound})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		rctx.template.ExecuteTemplate(w, "error", ErrorPageData{http.StatusInternalServerError})
		return
	}

	// Check if the user paid or canceled
	var payResult string
	payResults, _ := r.URL.Query()["r"]
	if len(payResults) == 1 {
		payResult = payResults[0]
	}

	// If this isn't the canceled page, and they haven't paid, 404
	if payResult != "cancel" && !paid {
		w.WriteHeader(http.StatusNotFound)
		rctx.template.ExecuteTemplate(w, "error", ErrorPageData{http.StatusNotFound})
		return
	}

	// Figure out social links
	shareLink := fmt.Sprintf("%s/?r=%s", rctx.appOrigin, publicToken)
	plural := "s"
	if numNets == 1 {
		plural = ""
	}
	mailBody := fmt.Sprintf("I just donated %d net%s: %s", numNets, plural, shareLink)
	mailLink := fmt.Sprintf("mailto:?subject=%%23NetCountdown&body=%s",
		url.PathEscape(mailBody))
	twitterLink := fmt.Sprintf("I just donated %d net%s %s #NetCountdown", numNets, plural, shareLink)
	data := CardPageData{
		PublicToken: publicToken,
		PayResult:   payResult,
		Paid:        paid,
		ShareLink:   shareLink,
		MailToLink:  mailLink,
		TwitterLink: twitterLink,
		CSRFField:   csrf.TemplateField(r),
	}

	// Render card page
	w.WriteHeader(http.StatusOK)
	rctx.template.ExecuteTemplate(w, "card", data)
}

func (rctx *RequestContext) cardDataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Look up the row by the public token (OK if not paid)
	var dbe common.DBQueueEntry
	const getMessageQuery = `SELECT * FROM messages WHERE publictoken=$1`
	err := rctx.pgClient.Get(&dbe, getMessageQuery, mux.Vars(r)["token"])
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		resp := common.CardResponse{
			Errors: []string{"Card not found"},
		}
		json.NewEncoder(w).Encode(resp)
		return
	} else if err != nil {
		internalServerErrorJSONResponse(w)
		return
	}

	// Extract QueueEntry from database entry
	qe := common.DBQueueEntryToQueueEntry(rctx.appOrigin, dbe)

	// Return the message
	w.WriteHeader(http.StatusOK)
	resp := common.CardResponse{
		Card: &qe,
	}
	json.NewEncoder(w).Encode(resp)
	return
}

func (rctx *RequestContext) leavingHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch the URL from the database by guid
	var url string
	linkQuery := `SELECT sociallink FROM messages WHERE publictoken=$1 AND ` +
		`socialtype='custom' AND paid='t'`
	err := rctx.pgClient.QueryRow(linkQuery, mux.Vars(r)["token"]).Scan(&url)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		rctx.template.ExecuteTemplate(w, "error", ErrorPageData{http.StatusNotFound})
		return
	} else if err != nil {
		log.Printf("leavingHandler: unexpected database error finding link: %v", err)
		w.WriteHeader(http.StatusNotFound)
		rctx.template.ExecuteTemplate(w, "error", ErrorPageData{http.StatusNotFound})
		return
	}

	// Render the interstitial page
	w.WriteHeader(http.StatusOK)
	rctx.template.ExecuteTemplate(w, "leaving", LeavingPageData{url})
}

func (rctx *RequestContext) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	rctx.template.ExecuteTemplate(w, "error", ErrorPageData{http.StatusNotFound})
}

func (rctx *RequestContext) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (rctx *RequestContext) thankYouHandler(w http.ResponseWriter, r *http.Request) {
	// Social media links
	shareLink := fmt.Sprintf("%s/", rctx.appOrigin)
	mailBody := fmt.Sprintf("I just donated some mosquito nets! %s", shareLink)
	twitterLink := fmt.Sprintf("I just donated some mosquito nets! %s #NetCountdown", shareLink)
	mailLink := fmt.Sprintf("mailto:?subject=%%23NetCountdown&body=%s",
		url.PathEscape(mailBody))
	data := ThankYouPageData{
		ShareLink:   shareLink,
		MailToLink:  mailLink,
		TwitterLink: twitterLink,
	}
	w.WriteHeader(http.StatusOK)
	rctx.template.ExecuteTemplate(w, "thankyou", data)
}

func (rctx *RequestContext) wsHandler(ws *websocket.Conn) {
	// Subscribe to broadcast messages
	idx, bcast := rctx.messageNotifier.subscribeClient()
	defer rctx.messageNotifier.unsubscribeClient(idx)
	defer ws.Close()

	// Send initial cached messages
	initialMessageKeys := []string{
		common.InitialNetsGivenKey,
		common.InitialQueueKey,
		common.InitialLeaderboardKey,
	}

	for _, key := range initialMessageKeys {
		resp, err := rctx.redisClient.Get(key).Result()
		if err != nil {
			log.Printf("wsHandler: failed to get %v from redis: %v", key, err)
			return
		}

		ws.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
		err = websocket.Message.Send(ws, resp)
		if err != nil {
			log.Printf("wsHandler: failed to send initial message: %v", err)
			return
		}
		ws.SetWriteDeadline(websocketNoTimeout)
	}

	// Configure read limit
	ws.MaxPayloadBytes = maxPayloadBytes

	// Read pongs
	go func() {
		var tmp [512]byte
		var err error
		for {
			ws.SetReadDeadline(time.Now().Add(websocketReadTimeout))
			_, err = ws.Read(tmp[:])
			if err != nil {
				ws.Close()
				return
			}
		}
	}()

	for {
		// Read next message from redis broadcast channel. Already serialized
		data, ok := <-bcast
		if !ok {
			break
		}

		// Forward to client
		if data != "" {
			ws.SetWriteDeadline(time.Now().Add(websocketWriteTimeout))
			err := websocket.Message.Send(ws, data)
			if err != nil {
				ws.Close()
				break
			}
			ws.SetWriteDeadline(websocketNoTimeout)
		}
	}
}

// Don't allow cross-origin connections
func ensureOrigin(origin string) func(*websocket.Config, *http.Request) error {
	return func(config *websocket.Config, req *http.Request) (err error) {
		originHeader := req.Header.Get("Origin")
		if originHeader != origin {
			return fmt.Errorf("invalid origin")
		}
		return err
	}
}

func main() {
	// Ensure required environment variables are set
	appOrigin := os.Getenv("MN_APP_ORIGIN")
	if appOrigin == "" {
		log.Fatalf("MN_APP_ORIGIN (application server origin) environment variable is required")
	}

	wsOrigin := os.Getenv("MN_WS_ORIGIN")
	if wsOrigin == "" {
		log.Fatalf("MN_WS_ORIGIN (websocket server origin) environment variable is required")
	}

	postgresURL := os.Getenv("MN_POSTGRES_URL")
	if postgresURL == "" {
		log.Fatalf("MN_POSTGRES_URL (postgres connection URL) environment variable is required")
	}

	redisAddr := os.Getenv("MN_REDIS_ADDR")
	if redisAddr == "" {
		log.Fatalf("MN_REDIS_ADDR (redis address) environment variable is required")
	}

	redisPassword := os.Getenv("MN_REDIS_PASS")

	csrfSecret := os.Getenv("MN_CSRF_SECRET")
	if len(csrfSecret) != 32 {
		log.Fatalf("MN_CSRF_SECRET (32-byte csrf secret) environment variable is required")
	}

	environment := os.Getenv("MN_ENVIRONMENT")
	if environment == "" {
		log.Fatalf("MN_ENVIRONMENT (deploy environment) environment variable is required")
	}

	stripeSecret := os.Getenv("MN_STRIPE_SECRET")
	if stripeSecret == "" {
		log.Fatalf("MN_STRIPE_SECRET (stripe secret token) environment variable is required")
	}

	stripePublicToken := os.Getenv("MN_STRIPE_PUBLIC")
	if stripePublicToken == "" {
		log.Fatalf("MN_STRIPE_PUBLIC (stripe public token) environment variable is required")
	}

	tokenPermutationSecret := os.Getenv("MN_TOKEN_PERM_SECRET")
	if tokenPermutationSecret == "" {
		log.Fatalf("MN_TOKEN_PERM_SECRET (56-byte message token permutation secret) environment variable is required")
	}

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

	// Wait for timeline server to finish populating redis
	for i = 0; i < numTries; i++ {
		_, err := redisClient.Get(common.StartupKey).Result()
		if err != nil {
			log.Printf("failed to get startup key from redis (try %d/%d): %v", i+1, numTries, err)
			time.Sleep(1 * time.Second)
			continue
		}
		log.Printf("redis initialized")
		break
	}
	if i == numTries {
		log.Fatalf("failed waiting for timeline to set up redis")
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

	// Construct & start notification service
	messageNotifier := MessageNotifier{
		redisClient: redisClient,
		subscribers: make(map[uint64](chan string)),
	}
	go messageNotifier.handleBroadcastMessages()

	// Are we a development environment?
	development := strings.HasPrefix(strings.ToLower(environment), "dev")

	// Construct request context. Handlers are methods on this.
	rctx := RequestContext{
		messageNotifier:        &messageNotifier,
		pgClient:               pgClient,
		redisClient:            redisClient,
		template:               template.Must(template.ParseGlob("template/*.tmpl")),
		appOrigin:              appOrigin,
		wsOrigin:               wsOrigin,
		stripePublicToken:      stripePublicToken,
		development:            development,
		tokenPermutationSecret: []byte(tokenPermutationSecret),
	}

	// Initialize app router
	ar := mux.NewRouter()
	tokRxp := "{token:[A-Z2-7]{13}}"
	ar.HandleFunc("/", rctx.homeHandler).Methods("GET")
	ar.HandleFunc("/send", rctx.sendHandler).Methods("GET")
	ar.HandleFunc(fmt.Sprintf("/send/%s", tokRxp), rctx.sendHandler).Methods("GET")
	ar.HandleFunc("/all", rctx.allHandler).Methods("GET")
	ar.HandleFunc("/all/{page:[0-9]+}", rctx.allHandler).Methods("GET")
	ar.HandleFunc("/messages/{page:[0-9]+}", rctx.messagesHandler).Methods("GET")
	ar.HandleFunc("/leaderboard", rctx.leaderboardHandler).Methods("GET")
	ar.HandleFunc("/leaderboard/{page:[0-9]+}", rctx.leaderboardHandler).Methods("GET")
	ar.HandleFunc("/rankings/{page:[0-9]+}", rctx.rankingsHandler).Methods("GET")
	ar.HandleFunc("/init.js", rctx.stripeInitHandler).Methods("GET")
	ar.HandleFunc(fmt.Sprintf("/card/%s", tokRxp), rctx.cardHandler).Methods("GET")
	ar.HandleFunc(fmt.Sprintf("/card/%s/data", tokRxp), rctx.cardDataHandler).Methods("GET")
	ar.HandleFunc(fmt.Sprintf("/leaving/%s", tokRxp), rctx.leavingHandler).Methods("GET")
	ar.HandleFunc(fmt.Sprintf("/pay/%s/retry", tokRxp), rctx.retryHandler).Methods("POST")
	ar.HandleFunc("/pay", rctx.payHandler).Methods("POST")
	ar.HandleFunc("/thankyou", rctx.thankYouHandler).Methods("GET")
	ar.HandleFunc("/health", rctx.healthHandler).Methods("GET")
	ar.PathPrefix("/static/").Handler(http.StripPrefix("/static/",
		http.FileServer(http.Dir("./static/"))))
	ar.NotFoundHandler = http.HandlerFunc(rctx.notFoundHandler)

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

	// Require secure CSRF in non-dev environment
	secureCsrf := !development
	csrfMiddleware := csrf.Protect([]byte(csrfSecret),
		csrf.Secure(secureCsrf),
		csrf.FieldName("tok"))

	// Configure app server
	appSrv := &http.Server{
		Handler:      secureMiddleware.Handler(csrfMiddleware(ar)),
		Addr:         appAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Start app server, don't block yet
	log.Printf("Starting server on %s", appAddress)
	go appSrv.ListenAndServe()

	// Configure websocket server + router (allow access from appOrigin)
	ws := websocket.Server{
		Handler:   rctx.wsHandler,
		Handshake: ensureOrigin(appOrigin),
	}
	wr := mux.NewRouter()
	wr.Handle("/ws", ws)
	wr.HandleFunc("/health", rctx.healthHandler).Methods("GET")
	wsSrv := &http.Server{
		Handler:      secureMiddleware.Handler(wr),
		Addr:         wsAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Starting server on %s", wsAddress)
	wsSrv.ListenAndServe()
}
