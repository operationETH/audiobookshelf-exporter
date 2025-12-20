package api

import (
    "encoding/json"
    "fmt"
    "net/http"
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
    LibraryID     string          `json:"libraryId"`
    UserID        string          `json:"userId"`
    MediaType     string          `json:"mediaType"`
    Duration      float64         `json:"duration"`      
    TimeListening float64         `json:"timeListening"` 
    Date          string          `json:"date"`          
    DayOfWeek     string          `json:"dayOfWeek"`     
    User          *SessionUser    `json:"user"`
    DeviceInfo    *SessionDevice  `json:"deviceInfo"`
    MediaMetadata *MediaMetadata  `json:"mediaMetadata"`
}

type SessionUser struct {
    ID       string `json:"id"`
    Username string `json:"username"`
}

type SessionDevice struct {
    ClientName string `json:"clientName"`
    Model      string `json:"model"`
}

type MediaMetadata struct {
    Title string `json:"title"`
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
    return 0, nil
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
