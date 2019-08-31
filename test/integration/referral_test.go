package send_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/justicz/giveanet/common"

	"github.com/stretchr/testify/require"
)

func TestReferralsPaid(t *testing.T) {
	const maxNets = 100
	const referralDepth = 100

	// Empty messages database
	clearDatabase(t)

	// Seed random
	rand.Seed(time.Now().UnixNano())

	// Keep tally of expected net scores
	var netScores []uint64

	// Build a long chain of referrals
	for i := 0; i < referralDepth; i++ {
		form := url.Values{}
		form.Set("wantmsg", "on")

		// Pick a random number of nets
		numNets := (uint64)(rand.Intn(maxNets) + 1)

		// Update expected net scores
		for j := range netScores {
			netScores[j] += numNets
		}

		// Add our donation
		netScores = append(netScores, numNets)

		form.Set("netslider", fmt.Sprintf("%d", numNets))
		form.Set("netnumbox", "76")
		form.Set("netquantitytype", "reg")
		form.Set("displayname", fmt.Sprintf("%d place", i + 1))
		form.Set("linktype", "twitter")
		form.Set("twittername", "foobar4u")
		form.Set("msg", "baz biz")

		// Get referred by last entry
		var cookies []*http.Cookie
		if i != 0 {
			refCode := getLastReferralCode(t)
			cookie := http.Cookie{
				Name:  "referral",
				Value: refCode,
			}
			cookies = append(cookies, &cookie)
		}

		// Generate random fake icon
		var fakeIcon [300]byte
		_, err := rand.Read(fakeIcon[:])
		require.NoError(t, err)
		form.Set("canvasdata", base64.StdEncoding.EncodeToString(fakeIcon[:]))

		// POST form
		payURL := fmt.Sprintf("%s/pay", appDevHost)
		resp, err := postForm(payURL, form, cookies)
		require.NoError(t, err)

		// Must succeed
		require.Equal(t, 200, resp.StatusCode)

		// Decode response
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		fr := common.PayFormResponse{}
		err = json.Unmarshal(bodyBytes, &fr)
		require.NoError(t, err)

		// Should contain token
		require.NotEmpty(t, fr.StripeSession)

		// Mark as paid (hit timeline webhook)
		form = url.Values{}
		form.Set("token", fr.StripeSession)
		webhookURL := fmt.Sprintf("%s/api/webhook/%s/stripe", timelineDevHost, webhookDevSecret)
		resp, err = postForm(webhookURL, form, nil)
		require.NoError(t, err)

		// Must succeed
		require.Equal(t, 200, resp.StatusCode)
	}

	// Check that all net scores are expected
	var entries []common.DBQueueEntry
	pgClient := getDB(t)
	err := pgClient.Select(&entries, "SELECT * from messages ORDER BY message_id ASC")
	require.NoError(t, err)
	require.Equal(t, len(entries), len(netScores))
	for i, entry := range entries {
		require.Equal(t, entry.NetPoints, netScores[i])
	}
}
