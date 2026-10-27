package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/skinnykaen/robbo_student_personal_account.git/package/models"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/notifications"
	"github.com/spf13/viper"
)

// Open edX notifications of a user, read through the LMS service API
// (openedx-platform lms/djangoapps/robbo_notifications/service_api.py) as the LK service account:
// OAuth2 client credentials of the app lk-service, JWT. Config: lmsNotifications.* (config/init.go).

const lmsServicePath = "/api/robbo/v1/notifications/service/"

// The bell must not hang on a slow LMS: the usecase shows LK notifications without the LMS ones.
var lmsHTTPClient = &http.Client{Timeout: 3 * time.Second}

var errLMSUnauthorized = errors.New("lms notifications: unauthorized")

type LMSGateway struct {
	baseURL      string
	clientID     string
	clientSecret string
	client       *http.Client

	mu      sync.Mutex
	token   string
	expires time.Time
}

// NewLMSGatewayFromConfig: the LMS source, or a disabled one when lmsNotifications is off or incomplete.
func NewLMSGatewayFromConfig() notifications.LMSSource {
	if !viper.GetBool("lmsNotifications.enabled") {
		return DisabledLMS{}
	}
	base := strings.TrimRight(strings.TrimSpace(viper.GetString("lmsNotifications.baseUrl")), "/")
	id := strings.TrimSpace(viper.GetString("lmsNotifications.clientId"))
	secret := viper.GetString("lmsNotifications.clientSecret")
	if base == "" || id == "" || secret == "" {
		log.Println("[notifications] LMS feed off: lmsNotifications.baseUrl / clientId / clientSecret not set")
		return DisabledLMS{}
	}
	return NewLMSGateway(base, id, secret, lmsHTTPClient)
}

func NewLMSGateway(baseURL, clientID, clientSecret string, client *http.Client) *LMSGateway {
	return &LMSGateway{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       client,
	}
}

func (g *LMSGateway) Enabled() bool { return true }

func (g *LMSGateway) accessToken(renew bool) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !renew && g.token != "" && time.Now().Before(g.expires) {
		return g.token, nil
	}
	response, err := g.client.PostForm(g.baseURL+"/oauth2/access_token", url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"token_type":    {"jwt"},
	})
	if err != nil {
		return "", fmt.Errorf("lms notifications: token: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lms notifications: token: HTTP %d", response.StatusCode)
	}
	var token struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil || token.AccessToken == "" {
		return "", errors.New("lms notifications: token: bad response")
	}
	g.token = token.AccessToken
	// A minute early, so a token never expires between the check and the call.
	g.expires = time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - time.Minute)
	return g.token, nil
}

// call sends one request; a 401 renews the token once (the LMS may have restarted with new keys).
func (g *LMSGateway) call(method, path string, query url.Values, body, out interface{}) error {
	err := g.callOnce(method, path, query, body, out, false)
	if errors.Is(err, errLMSUnauthorized) {
		err = g.callOnce(method, path, query, body, out, true)
	}
	return err
}

func (g *LMSGateway) callOnce(method, path string, query url.Values, body, out interface{}, renew bool) error {
	token, err := g.accessToken(renew)
	if err != nil {
		return err
	}
	target := g.baseURL + lmsServicePath + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequest(method, target, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "JWT "+token)
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := g.client.Do(request)
	if err != nil {
		return fmt.Errorf("lms notifications: %s %s: %w", method, path, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return errLMSUnauthorized
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("lms notifications: %s %s: HTTP %d", method, path, response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(out)
}

type lmsItem struct {
	ID         int64   `json:"id"`
	Type       string  `json:"type"`
	Label      string  `json:"label"`
	Text       string  `json:"text"`
	CourseName string  `json:"course_name"`
	URL        string  `json:"url"`
	Created    string  `json:"created"`
	Read       bool    `json:"read"`
	ReadAt     *string `json:"read_at"`
}

func parseTime(value string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, value)
	return t, err == nil
}

// feedItem: an LMS notification in the bell's shape. Title = what happened («Задание проверено»),
// body = the text itself, as in the LMS bell.
func feedItem(item lmsItem) models.NotificationFeedItemHTTP {
	created, _ := parseTime(item.Created)
	row := models.NotificationFeedItemHTTP{
		FeedKind:   "lms",
		ID:         strconv.FormatInt(item.ID, 10),
		Title:      item.Label,
		Body:       item.Text,
		Kind:       item.Type,
		Severity:   "INFO",
		Source:     "lms",
		CreatedAt:  created,
		CourseName: item.CourseName,
	}
	if item.Read {
		readAt := created
		if item.ReadAt != nil {
			if t, ok := parseTime(*item.ReadAt); ok {
				readAt = t
			}
		}
		row.ReadAt = &readAt
	}
	if item.URL != "" {
		link := item.URL
		row.ActionURL = &link
	}
	return row
}

func lmsUserID(userID string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(userID), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("lms notifications: user id %q is not an LMS id", userID)
	}
	return id, nil
}

func (g *LMSGateway) Feed(userID string, limit int) (notifications.LMSFeed, error) {
	if _, err := lmsUserID(userID); err != nil {
		return notifications.LMSFeed{}, err
	}
	var data struct {
		Enabled bool      `json:"enabled"`
		Items   []lmsItem `json:"items"`
		Links   struct {
			Feed     string `json:"feed"`
			Settings string `json:"settings"`
		} `json:"links"`
	}
	query := url.Values{"user_id": {userID}, "limit": {strconv.Itoa(limit)}}
	if err := g.call(http.MethodGet, "feed/", query, nil, &data); err != nil {
		return notifications.LMSFeed{}, err
	}
	feed := notifications.LMSFeed{Enabled: data.Enabled}
	if !data.Enabled {
		return feed, nil
	}
	for _, item := range data.Items {
		feed.Items = append(feed.Items, feedItem(item))
	}
	if data.Links.Feed != "" {
		feed.Links = &models.LMSNotificationLinksHTTP{Feed: data.Links.Feed, Settings: data.Links.Settings}
	}
	return feed, nil
}

func (g *LMSGateway) UnreadCount(userID string) (int64, error) {
	if _, err := lmsUserID(userID); err != nil {
		return 0, err
	}
	var data struct {
		Enabled bool  `json:"enabled"`
		Unread  int64 `json:"unread"`
	}
	if err := g.call(http.MethodGet, "unread-count/", url.Values{"user_id": {userID}}, nil, &data); err != nil {
		return 0, err
	}
	if !data.Enabled {
		return 0, nil
	}
	return data.Unread, nil
}

func (g *LMSGateway) MarkRead(userID string, ids []int64) error {
	id, err := lmsUserID(userID)
	if err != nil {
		return err
	}
	var out struct{}
	return g.call(http.MethodPost, "read/", nil, map[string]interface{}{"user_id": id, "ids": ids}, &out)
}

func (g *LMSGateway) MarkAllRead(userID string) error {
	id, err := lmsUserID(userID)
	if err != nil {
		return err
	}
	var out struct{}
	return g.call(http.MethodPost, "read/", nil, map[string]interface{}{"user_id": id, "all": true}, &out)
}

// DisabledLMS: no LMS feed (lmsNotifications off): the bell shows LK notifications only.
type DisabledLMS struct{}

func (DisabledLMS) Enabled() bool { return false }

func (DisabledLMS) Feed(string, int) (notifications.LMSFeed, error) {
	return notifications.LMSFeed{}, nil
}

func (DisabledLMS) UnreadCount(string) (int64, error) { return 0, nil }

func (DisabledLMS) MarkRead(string, []int64) error { return nil }

func (DisabledLMS) MarkAllRead(string) error { return nil }
