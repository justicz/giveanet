package send_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/stretchr/testify/require"
)

const appDevHost = "http://millionnets-app-proxy"
const timelineDevHost = "http://millionnets-timeline-proxy"

var csrfDevSecret = os.Getenv("MN_CSRF_SECRET")
var webhookDevSecret = os.Getenv("MN_WEBHOOK_SECRET")

const csrfField = "tok"

var dbHandle *sqlx.DB

func getDB(t *testing.T) *sqlx.DB {
	if dbHandle != nil {
		return dbHandle
	}
	postgresURL := os.Getenv("MN_POSTGRES_URL")
	if !strings.Contains(postgresURL, "mndev") {
		require.FailNow(t, "don't run this in production, silly")
	}
	var err error
	dbHandle, err = sqlx.Open("postgres", postgresURL)
	require.NoError(t, err)
	return dbHandle
}

func clearDatabase(t *testing.T) {
	pgClient := getDB(t)
	_, err := pgClient.Exec("DELETE from messages")
	require.NoError(t, err)
}

func getLastReferralCode(t *testing.T) string {
	var refID *string
	const q = `SELECT publictoken FROM messages ORDER BY message_id DESC LIMIT 1`
	pgClient := getDB(t)
	err := pgClient.QueryRow(q).Scan(&refID)
	require.NoError(t, err)
	require.NotEmpty(t, refID)
	return *refID
}

func postForm(url string, form url.Values, cookies []*http.Cookie) (*http.Response, error) {
	// Add CSRF token to form data
	cookie, token, err := getCSRFInfo()
	form.Set(csrfField, token)

	// Build request
	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	// Add CSRF cookie
	req.AddCookie(cookie)

	// Add extra cookies
	for _, c := range cookies {
		req.AddCookie(c)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Do request
	client := &http.Client{}
	return client.Do(req)
}

func getCSRFInfo() (*http.Cookie, string, error) {
	router := mux.NewRouter()

	// Stub handler that just returns a token
	h := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, csrf.Token(r))
	}
	router.HandleFunc("/", h)

	// Initialize csrf library
	mw := csrf.Protect([]byte(csrfDevSecret),
		csrf.Secure(false),
		csrf.FieldName(csrfField))

	// Record a response
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	mwRouter := mw(router)
	mwRouter.ServeHTTP(recorder, req)
	resp := recorder.Result()

	// Read cookie out of response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Parse Set-Cookie
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		return nil, "", fmt.Errorf("unexpected cookies: %+v", cookies)
	}

	// Return token and cookie
	cookie := resp.Cookies()[0]
	token := string(body)
	return cookie, token, nil
}
