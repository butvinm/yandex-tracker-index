package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"net/http"
	"net/url"
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

type AttachmentInfo struct {
	Self    string `json:"self"`
	Id      string `json:"id"`
	Display string `json:"display"`
}

type Attachment struct {
	Self      string `json:"self"`
	Id        string `json:"id"`
	Name      string `json:"name"`
	Content   string `json:"content"`
	Thumbnail string `json:"thumbnail"`
	CreatedBy *User  `json:"createdBy"`
	CreatedAt Date   `json:"createdAt"`
	Mimetype  string `json:"mimetype"`
	Size      int    `json:"size"`
}

type Comment struct {
	Self        string           `json:"self"`
	Id          int              `json:"id"`
	LongId      string           `json:"longId"`
	Text        string           `json:"text"`
	TextHtml    string           `json:"textHtml"`
	CreatedBy   *User            `json:"createdBy"`
	UpdatedBy   *User            `json:"updatedBy"`
	CreatedAt   Date             `json:"createdAt"`
	UpdatedAt   Date             `json:"updatedAt"`
	Version     int              `json:"version"`
	Type        string           `json:"type"`
	Transport   string           `json:"transport"`
	Attachments []AttachmentInfo `json:"attachments"`
}

type Issue struct {
	Self                 string           `json:"self"`
	ID                   string           `json:"id"`
	Key                  string           `json:"key"`
	Version              int              `json:"version"`
	LastCommentUpdatedAt Date             `json:"lastCommentUpdatedAt,omitempty"`
	Summary              string           `json:"summary"`
	Parent               *ParentLink      `json:"parent,omitempty"`
	Aliases              []string         `json:"aliases,omitempty"`
	UpdatedBy            *User            `json:"updatedBy,omitempty"`
	Description          string           `json:"description,omitempty"`
	Sprint               []SprintInfo     `json:"sprint,omitempty"`
	Type                 *IssueType       `json:"type"`
	Priority             *PriorityInfo    `json:"priority"`
	CreatedAt            Date             `json:"createdAt"`
	Followers            []User           `json:"followers,omitempty"`
	CreatedBy            *User            `json:"createdBy"`
	Votes                int              `json:"votes"`
	Assignee             *User            `json:"assignee,omitempty"`
	Project              *ProjectDetails  `json:"project,omitempty"`
	Queue                *QueueInfo       `json:"queue"`
	UpdatedAt            Date             `json:"updatedAt"`
	Status               *StatusInfo      `json:"status"`
	PreviousStatus       *StatusInfo      `json:"previousStatus,omitempty"`
	Favorite             bool             `json:"favorite"`
	Tags                 []string         `json:"tags,omitempty"`
	Attachments          []AttachmentInfo `json:"attachments"`
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
		BaseURL: "https://api.tracker.yandex.net/",
		Token:   token,
		OrgID:   orgID,
		Client:  &client,
	}
}

func (t *YandexTrackerClient) newRequest(ctx context.Context, method, pathStr string, queryParams url.Values, body io.Reader) (*http.Request, error) {
	rel := &url.URL{Path: pathStr}
	if queryParams != nil {
		rel.RawQuery = queryParams.Encode()
	}

	u, err := url.Parse(t.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	u = u.ResolveReference(rel)
	log.Println("url: ", u)
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "OAuth "+t.Token)
	req.Header.Set("X-Org-ID", t.OrgID)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (t *YandexTrackerClient) do(req *http.Request, target interface{}) error {
	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return APIError{StatusCode: resp.StatusCode, Body: fmt.Appendf(nil, "failed to read error body: %v", readErr)}
		}
		return APIError{StatusCode: resp.StatusCode, Body: respBody}
	}

	if target != nil {
		if b, ok := target.(*[]byte); ok {
			*b, err = io.ReadAll(resp.Body)
			return err
		}
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("error decoding response: %w", err)
		}
	}
	return nil
}

func (t *YandexTrackerClient) GetIssuesCount(ctx context.Context) (int, error) {
	req, err := t.newRequest(ctx, "GET", "/v2/issues/_count", nil, nil)
	if err != nil {
		return 0, err
	}

	var count float64
	if err := t.do(req, &count); err != nil {
		return 0, err
	}
	return int(count), nil
}

func (t *YandexTrackerClient) ListIssues(ctx context.Context, page int, perPage int) ([]*Issue, error) {
	params := url.Values{}
	params.Add("page", fmt.Sprintf("%d", page))
	params.Add("perPage", fmt.Sprintf("%d", perPage))
	params.Add("expand", "attachments")
	req, err := t.newRequest(ctx, "GET", "/v2/issues", params, nil)
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := t.do(req, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}
