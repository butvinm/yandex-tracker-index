package main

import (
	"encoding/json"
	"fmt"
	"io"

	// "io"

	"net/http"
)

// Format: "2017-07-18T13:33:44.291+0000"
type Date string

type User struct {
	Self        string `json:"self"`
	ID          string `json:"id"`
	Display     string `json:"display"`
	PassportUID int64  `json:"passportUid,omitempty"`
	CloudUID    string `json:"cloudUid,omitempty"`
}

type ParentLink struct {
	Self    string `json:"self"`
	ID      string `json:"id"`
	Key     string `json:"key"`
	Display string `json:"display"`
}

type SprintInfo struct {
	Self    string `json:"self"`
	ID      string `json:"id"`
	Display string `json:"display"`
}

type IssueType struct {
	Self    string `json:"self"`
	ID      string `json:"id"`
	Key     string `json:"key"`
	Display string `json:"display"`
}

type PriorityInfo struct {
	Self    string `json:"self"`
	ID      string `json:"id"`
	Key     string `json:"key"`
	Display string `json:"display"`
}

type ProjectInfo struct {
	Self    string `json:"self"`
	ID      string `json:"id"`
	Display string `json:"display"`
}

type ProjectDetails struct {
	Primary   *ProjectInfo  `json:"primary"`
	Secondary []ProjectInfo `json:"secondary"`
}

type QueueInfo struct {
	Self    string `json:"self"`
	ID      string `json:"id"`
	Key     string `json:"key"`
	Display string `json:"display"`
}

type StatusInfo struct {
	Self    string `json:"self"`
	ID      string `json:"id"`
	Key     string `json:"key"`
	Display string `json:"display"`
}

type Issue struct {
	Self                 string          `json:"self"`
	ID                   string          `json:"id"`
	Key                  string          `json:"key"`
	Version              int             `json:"version"`
	LastCommentUpdatedAt Date            `json:"lastCommentUpdatedAt,omitempty"`
	Summary              string          `json:"summary"`
	Parent               *ParentLink     `json:"parent,omitempty"`
	Aliases              []string        `json:"aliases,omitempty"`
	UpdatedBy            *User           `json:"updatedBy,omitempty"`
	Description          string          `json:"description,omitempty"`
	Sprint               []SprintInfo    `json:"sprint,omitempty"`
	Type                 *IssueType      `json:"type"`
	Priority             *PriorityInfo   `json:"priority"`
	CreatedAt            Date            `json:"createdAt"`
	Followers            []User          `json:"followers,omitempty"`
	CreatedBy            *User           `json:"createdBy"`
	Votes                int             `json:"votes"`
	Assignee             *User           `json:"assignee,omitempty"`
	Project              *ProjectDetails `json:"project,omitempty"`
	Queue                *QueueInfo      `json:"queue"`
	UpdatedAt            Date            `json:"updatedAt"`
	Status               *StatusInfo     `json:"status"`
	PreviousStatus       *StatusInfo     `json:"previousStatus,omitempty"`
	Favorite             bool            `json:"favorite"`
	Tags                 []string        `json:"tags,omitempty"`
}

type YandexTrackerClient struct {
	BaseURL string
	Token   string
	OrgID   string
	Client  *http.Client
}

type APIError struct {
	StatusCode int
	Body       []byte
}

func (e APIError) Error() string {
	return fmt.Sprintf("Yandex Tracker API Error %d: %s", e.StatusCode, string(e.Body))
}

func NewClient(token string, orgID string) YandexTrackerClient {
	client := http.Client{}
	return YandexTrackerClient{
		BaseURL: "https://api.tracker.yandex.net/v2",
		Token:   token,
		OrgID:   orgID,
		Client:  &client,
	}
}

func (t *YandexTrackerClient) GetIssuesCount() (int, error) {
	// POST /v3/issues/_count
	// Host: api.tracker.yandex.net
	// Authorization: OAuth <OAuth-токен>
	// X-Org-ID или X-Cloud-Org-ID: <идентификатор_организации>
	u := fmt.Sprintf("%s/issues/_count", t.BaseURL)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Authorization", "OAuth "+t.Token)
	req.Header.Add("X-Org-ID", t.OrgID)

	resp, err := t.Client.Do(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, APIError{StatusCode: resp.StatusCode, Body: body}
	}

	var count float64
	if err := json.NewDecoder(resp.Body).Decode(&count); err != nil {
		return 0, err
	}
	return int(count), nil
}

func (t *YandexTrackerClient) ListIssues(page int, perPage int) ([]*Issue, error) {
	u := fmt.Sprintf("%s/issues?page=%d&perPage=%d", t.BaseURL, page, perPage)

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "OAuth "+t.Token)
	req.Header.Add("X-Org-ID", t.OrgID)

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, APIError{StatusCode: resp.StatusCode, Body: body}
	}

	var issues []*Issue
	if err = json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}
	return issues, nil
}
