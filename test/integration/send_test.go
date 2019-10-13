package send_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"testing"

	"github.com/justicz/giveanet/common"

	"github.com/stretchr/testify/require"
)

var mapKeys []string

func init() {
	for k := range common.AllowedCountries {
		mapKeys = append(mapKeys, k)
	}
}

func TestSendNetBasic(t *testing.T) {
	const maxNets = 100

	// Empty messages database
	clearDatabase(t)

	for i := 0; i < 50; i++ {
		form := url.Values{}
		form.Set("wantmsg", "on")

		// Pick a random number of nets
		form.Set("netslider", fmt.Sprintf("%d", rand.Intn(maxNets)+1))
		form.Set("netnumbox", "76")
		form.Set("netquantitytype", "reg")
		form.Set("displayname", "foo bar")
		form.Set("linktype", "twitter")
		form.Set("twittername", "foobar4u")
		form.Set("msg", "baz biz")
		form.Set("country", mapKeys[rand.Intn(len(mapKeys))])

		// Generate random fake icon
		var fakeIcon [300]byte
		_, err := rand.Read(fakeIcon[:])
		require.NoError(t, err)
		form.Set("canvasdata", base64.StdEncoding.EncodeToString(fakeIcon[:]))

		// POST form
		payURL := fmt.Sprintf("%s/pay", appDevHost)
		resp, err := postForm(payURL, form, nil)
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
}
