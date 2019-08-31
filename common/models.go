package common

import (
	"time"
)

const SocialLinkTypeNowhere = "nowhere"
const SocialLinkTypeTwitter = "twitter"
const SocialLinkTypeInstagram = "instagram"
const SocialLinkTypeCustom = "custom"

type DBQueueEntry struct {
	MessageID          uint64     `json:"message_id,omitempty" db:"message_id"`
	Referral           *uint64    `json:"-" db:"referral"`
	PublicToken        string     `json:"publictoken,omitempty" db:"publictoken"`
	Timeline           *uint64    `json:"timeline,omitempty" db:"timeline"`
	Message            *string    `json:"message,omitempty" db:"message"`
	NetPoints          uint64     `json:"netpoints,omitempty" db:"netpoints"`
	DisplayName        *string    `json:"displayname,omitempty" db:"displayname"`
	SocialType         string     `json:"socialtype,omitempty" db:"socialtype"`
	SocialLink         *string    `json:"sociallink,omitempty" db:"sociallink"`
	NumNets            uint64     `json:"numnets,omitempty" db:"numnets"`
	ImgKind            string     `json:"imgkind,omitempty" db:"imgkind"`
	ImgData            []byte     `json:"imgdata,omitempty" db:"imgdata"`
	Created            *time.Time `json:"created,omitempty" db:"created"`
	PaidTime           *time.Time `json:"paidtime,omitempty" db:"paidtime"`
	StripeSessionToken string     `json:"-" db:"stripesessiontoken"`
	Paid               bool       `json:"paid" db:"paid"`
	Played             bool       `json:"played" db:"played"`
	Rank               *uint64    `json:"rank" db:"rank"`
}
