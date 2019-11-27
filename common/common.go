package common

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

const NetCostCents = 200

const PingChannel = "ping"
const NetsGivenUpdateChannel = "countupdated"
const LeaderboardUpdateChannel = "leaderupdate"
const QueueUpdateChannel = "queueupdate"

const InitialNetsGivenKey = "initialnetsgiven"
const InitialQueueKey = "initialqueue"
const InitialLeaderboardKey = "initialleaderboard"
const InitialGoalKey = "initialgoal"
const StartupKey = "startup"

const PagesToCache = 10
const LeaderboardCachePageFmt = "loaderboardpg%d"
const CountryLeaderboardCachePageFmt = "countryloaderboardpg%d"
const QueueCachePageFmt = "queuepg%d"
const LeaderboardPageSize = 11
const QueuePageSize = 7

var Goals = []Goal{
	Goal{Name: "Base Goal", NumNets: 10000, Nines: "9,999"},
	Goal{Name: "Stretch Goal", NumNets: 50000, Nines: "49,999"},
	Goal{Name: "Mega Goal", NumNets: 100000, Nines: "99,999"},
	Goal{Name: "Ultra Goal", NumNets: 500000, Nines: "499,999"},
	Goal{Name: "Million Goal", NumNets: 1000000, Nines: "999,999"},
}

var DefaultGoal = Goals[0]

func GetNetsGivenResponse(db *sqlx.DB) (resp Response, err error) {
	// Get latest net count
	var netsGiven uint64
	netsGivenQuery := `SELECT COALESCE(SUM(numnets), 0) FROM messages WHERE paid=true`
	err = db.QueryRow(netsGivenQuery).Scan(&netsGiven)
	if err != nil {
		log.Printf("GetNetsGivenResponse: unexpected database error computing sum: %v", err)
		return
	}

	// Fill in response
	resp = Response{
		Type: NetsGivenResponseType,
		NetsGivenResponse: &(NetsGivenResponse{
			NetsGiven: netsGiven,
		}),
	}
	return
}

func GetQueueResponse(db *sqlx.DB, origin string, page uint64) (resp Response, err error) {
	// Fetch requested page of messages. TODO(maxj): Use something better than
	// LIMIT/OFFSET if this is too inefficient
	var dbEntries []DBQueueEntry
	const queueQuery = `SELECT * FROM messages WHERE paid='t' AND played='t' ` +
		`ORDER BY timeline DESC OFFSET $1 LIMIT $2`
	err = db.Select(&dbEntries, queueQuery, QueuePageSize*page, QueuePageSize)
	if err != nil {
		log.Printf("GetQueueResponse: failed to get []DBQueueEntry: %v", err)
		return
	}

	// Build slice of last played QueuePageSize messages in the order that
	// they were played
	var entries []QueueEntry
	for i, entry := range dbEntries {
		qe := DBQueueEntryToQueueEntry(origin, entry)
		setRank(&qe, QueuePageSize*page, uint64(i))
		entries = append(entries, qe)
	}

	// Fill in the response
	resp = Response{
		Type: QueueResponseType,
		QueueResponse: &(QueueResponse{
			Entries: entries,
		}),
	}
	return
}

func GetCountryLeaderboardResponse(db *sqlx.DB, origin string, page uint64) (resp Response, err error) {
	// Fetch requested page of leaderboard. TODO(maxj): Use something better than
	// LIMIT/OFFSET if this is too inefficient. Note that for countries we sum
	// total net donations, not netpoints, so as not to double count
	var dbEntries []DBCountryLeaderboardEntry
	const lbQuery = `SELECT country, SUM(numnets) as netpoints FROM messages WHERE ` +
		`paid='t' GROUP BY country ORDER BY netpoints DESC OFFSET $1 LIMIT $2`
	err = db.Select(&dbEntries, lbQuery, LeaderboardPageSize*page, LeaderboardPageSize)
	if err != nil {
		log.Printf("GetCountryLeaderboardResponse: failed to get []DBQueueEntry: %v", err)
		return
	}

	var entries []QueueEntry
	for i, dbe := range dbEntries {
		if dbe.Country == nil || *dbe.Country == "none" {
			empty := ""
			dbe.Country = &empty
		}
		qe := DBCountryLeaderboardEntryToQueueEntry(dbe)
		setRank(&qe, LeaderboardPageSize*page, uint64(i))
		entries = append(entries, qe)
	}

	// Fill in response
	resp = Response{
		Type: LeaderboardResponseType,
		LeaderboardResponse: &(LeaderboardResponse{
			Entries: entries,
		}),
	}
	return
}

func GetLeaderboardResponse(db *sqlx.DB, origin string, page uint64) (resp Response, err error) {
	// Fetch requested page of leaderboard. TODO(maxj): Use something better than
	// LIMIT/OFFSET if this is too inefficient
	var dbEntries []DBQueueEntry
	const lbQuery = `SELECT * FROM messages WHERE paid='t' ORDER BY netpoints ` +
		`DESC OFFSET $1 LIMIT $2`
	err = db.Select(&dbEntries, lbQuery, LeaderboardPageSize*page, LeaderboardPageSize)
	if err != nil {
		log.Printf("GetLeaderboardResponse: failed to get []DBQueueEntry: %v", err)
		return
	}

	var entries []QueueEntry
	for i, dbe := range dbEntries {
		qe := DBQueueEntryToQueueEntry(origin, dbe)
		setRank(&qe, LeaderboardPageSize*page, uint64(i))
		entries = append(entries, qe)
	}

	// Fill in response
	resp = Response{
		Type: LeaderboardResponseType,
		LeaderboardResponse: &(LeaderboardResponse{
			Entries: entries,
		}),
	}
	return
}

func setRank(qe *QueueEntry, offset, idx uint64) {
	rank := offset + idx + 1
	qe.Rank = &rank
}

func DBCountryLeaderboardEntryToQueueEntry(le DBCountryLeaderboardEntry) (qe QueueEntry) {
	qe.Points = le.NetPoints
	if le.Country != nil {
		countryName := AllowedCountries[*le.Country]
		qe.Name = &countryName
	}
	qe.Country = le.Country
	return
}

func DBQueueEntryToQueueEntry(origin string, dbe DBQueueEntry) (qe QueueEntry) {
	// Because we have a "retry payment" page, it's possible we show a client a nil
	// timeline field
	if dbe.Timeline != nil {
		qe.TimelineIdx = int64(*dbe.Timeline)
	} else {
		qe.TimelineIdx = -1
	}

	// Add interstitial for custom links and expand usernames. We don't validate
	// usernames super hard (they can have / and \n, etc.)
	if dbe.SocialLink != nil {
		switch dbe.SocialType {
		case SocialLinkTypeCustom:
			tmp := fmt.Sprintf("%s/leaving/%s", origin, dbe.PublicToken)
			qe.Link = &tmp
		case SocialLinkTypeTwitter:
			tmp := fmt.Sprintf("https://twitter.com/%s", *dbe.SocialLink)
			qe.Link = &tmp
		case SocialLinkTypeInstagram:
			tmp := fmt.Sprintf("https://www.instagram.com/%s", *dbe.SocialLink)
			qe.Link = &tmp
		case SocialLinkTypeNowhere:
			qe.Link = nil
		default:
			log.Printf("Unexpected social type: %v", dbe.SocialType)
			qe.Link = nil
		}
	}

	// For now all images are the same encoding
	encodedIcon := base64.StdEncoding.EncodeToString(dbe.ImgData)
	qe.Icon = &encodedIcon

	// Some fields we can just copy over
	qe.Nets = dbe.NumNets
	qe.Points = dbe.NetPoints
	qe.Name = dbe.DisplayName
	qe.Message = dbe.Message
	qe.Country = dbe.Country
	return
}
