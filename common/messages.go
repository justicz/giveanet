package common

const (
	QueueResponseType       = "q"
	LeaderboardResponseType = "l"
	NetsGivenResponseType   = "n"
	PingResponseType        = "p"
)

/*
 * Websocket Responses
 */

type Response struct {
	Type                string               `json:"type"`
	NetsGivenResponse   *NetsGivenResponse   `json:"nr,omitempty"`
	PingResponse        *PingResponse        `json:"pr,omitempty"`
	QueueResponse       *QueueResponse       `json:"qr,omitempty"`
	LeaderboardResponse *LeaderboardResponse `json:"lr,omitempty"`
}

type QueueResponse struct {
	Entries []QueueEntry `json:"q"`
}

type LeaderboardResponse struct {
	Entries []QueueEntry `json:"l"`
}

type NetsGivenResponse struct {
	NetsGiven uint64 `json:"ng"`
}

type PingResponse struct{}

/*
 * HTTP Responses
 */

type PayFormResponse struct {
	Errors        []string `json:"errors,omitempty"`
	StripeSession string   `json:"stripe,omitempty"`
}

type CardResponse struct {
	Errors []string    `json:"errors,omitempty"`
	Card   *QueueEntry `json:"card,omitempty"`
}

/*
 * Common
 */

type QueueEntry struct {
	TimelineIdx int64   `json:"idx"`
	Queue       string  `json:"q"`
	Name        *string `json:"name,omitempty"`
	Country     *string `json:"country,omitempty"`
	Link        *string `json:"link,omitempty"`
	Message     *string `json:"msg,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	Nets        uint64  `json:"nets"`
	Points      uint64  `json:"points"`
	Rank        *uint64 `json:"rank,omitempty"`
}

type CountUpdated struct {
	NetsGiven uint64 `json:"ng"`
}
