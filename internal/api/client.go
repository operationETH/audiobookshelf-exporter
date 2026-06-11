package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	apiKey  string
	httpc   *http.Client
}

type Library struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LibraryDetail struct {
	TotalItems int `json:"totalItems"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type SessionsResponse struct {
	Total        int       `json:"total"`
	NumPages     int       `json:"numPages"`
	Page         int       `json:"page"`
	ItemsPerPage int       `json:"itemsPerPage"`
	Sessions     []Session `json:"sessions"`
}

type Session struct {
	ID            string         `json:"id"`
	LibraryID     string         `json:"libraryId"`
	UserID        string         `json:"userId"`
	MediaType     string         `json:"mediaType"`
	DisplayTitle  string         `json:"displayTitle"`
	DisplayAuthor string         `json:"displayAuthor"`
	Duration      float64        `json:"duration"`
	TimeListening float64        `json:"timeListening"`
	StartTime     float64        `json:"startTime"`
	CurrentTime   float64        `json:"currentTime"`
	StartedAt     int64          `json:"startedAt"`
	UpdatedAt     int64          `json:"updatedAt"`
	Date          string         `json:"date"`
	DayOfWeek     string         `json:"dayOfWeek"`
	User          *SessionUser   `json:"user"`
	DeviceInfo    *SessionDevice `json:"deviceInfo"`
	MediaMetadata *MediaMetadata `json:"mediaMetadata"`
}

type SessionUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type SessionDevice struct {
	ClientName string `json:"clientName"`
	Model      string `json:"model"`
	DeviceName string `json:"deviceName"`
}

type MediaMetadata struct {
	Title         string `json:"title"`
	AuthorName    string `json:"authorName"`
	SeriesName    string `json:"seriesName"`
	PublishedYear string `json:"publishedYear"`
}

type OnlineResponse struct {
	UsersOnline  []User    `json:"usersOnline"`
	OpenSessions []Session `json:"openSessions"`
}

type LibraryItemsResponse struct {
	Results []LibraryItem `json:"results"`
	Total   int           `json:"total"`
	Limit   int           `json:"limit"`
	Page    int           `json:"page"`
}

type LibraryItem struct {
	ID        string           `json:"id"`
	LibraryID string           `json:"libraryId"`
	MediaType string           `json:"mediaType"`
	AddedAt   int64            `json:"addedAt"`
	Media     LibraryItemMedia `json:"media"`
}

type LibraryItemMedia struct {
	Metadata MediaMetadata `json:"metadata"`
	Duration float64       `json:"duration"`
	Size     float64       `json:"size"`
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpc:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) get(path string, target interface{}) error {
	req, _ := http.NewRequest("GET", c.baseURL+path, nil)
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("bad status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func (c *Client) Libraries() ([]Library, error) {
	var wrap struct {
		Libraries []Library `json:"libraries"`
	}
	err := c.get("/api/libraries", &wrap)
	return wrap.Libraries, err
}

func (c *Client) LibraryDetail(id string) (*LibraryDetail, error) {
	var d LibraryDetail
	err := c.get("/api/libraries/"+id+"/stats", &d)
	return &d, err
}

func (c *Client) Users() ([]User, error) {
	var wrap struct {
		Users []User `json:"users"`
	}
	err := c.get("/api/users", &wrap)
	return wrap.Users, err
}

func (c *Client) ActiveStreamsCount() (int, error) {
	openSessions, err := c.OpenSessions()
	if err != nil {
		return 0, err
	}
	return len(openSessions), nil
}

func (c *Client) OpenSessions() ([]Session, error) {
	var resp OnlineResponse
	err := c.get("/api/users/online", &resp)
	return resp.OpenSessions, err
}

func (c *Client) RecentSessions(limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 10
	}

	var resp SessionsResponse
	path := fmt.Sprintf("/api/sessions?sort=updatedAt&desc=1&limit=%d", limit)
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	return resp.Sessions, nil
}

func (c *Client) RecentLibraryItems(libraryID string, limit int) ([]LibraryItem, error) {
	if limit <= 0 {
		limit = 10
	}

	var resp LibraryItemsResponse
	path := fmt.Sprintf("/api/libraries/%s/items?limit=%d&sort=addedAt&desc=1", url.PathEscape(libraryID), limit)
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

func (c *Client) Sessions() ([]Session, error) {
	var all []Session
	page := 0

	for {
		var resp SessionsResponse
		path := fmt.Sprintf("/api/sessions?page=%d", page)
		if err := c.get(path, &resp); err != nil {
			return all, err
		}

		all = append(all, resp.Sessions...)

		page++
		if page >= resp.NumPages {
			break
		}
		if page > 200 {
			break
		}
	}

	return all, nil
}
