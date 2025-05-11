package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"net/http"
	"net/url"
)

const MaxPerPage = 50

type Datetime time.Time

func (d *Datetime) UnmarshalJSON(b []byte) (err error) {
	date, err := time.Parse(`"2006-01-02T15:04:05.000-0700"`, string(b))
	if err != nil {
		return err
	}
	*d = Datetime(date)
	return nil
}

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
	ID      string `json:"id"`
	Display string `json:"display"`
}

type Attachment struct {
	Self      string   `json:"self"`
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Content   string   `json:"content"`
	Thumbnail string   `json:"thumbnail"`
	CreatedBy *User    `json:"createdBy"`
	CreatedAt Datetime `json:"createdAt"`
	Mimetype  string   `json:"mimetype"`
	Size      int      `json:"size"`
}

type Comment struct {
	Self        string           `json:"self"`
	ID          int              `json:"id"`
	LongId      string           `json:"longId"`
	Text        string           `json:"text"`
	TextHtml    string           `json:"textHtml"`
	CreatedBy   *User            `json:"createdBy"`
	UpdatedBy   *User            `json:"updatedBy"`
	CreatedAt   Datetime         `json:"createdAt"`
	UpdatedAt   Datetime         `json:"updatedAt"`
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
	LastCommentUpdatedAt Datetime         `json:"lastCommentUpdatedAt,omitempty"`
	Summary              string           `json:"summary"`
	Parent               *ParentLink      `json:"parent,omitempty"`
	Aliases              []string         `json:"aliases,omitempty"`
	UpdatedBy            *User            `json:"updatedBy,omitempty"`
	Description          string           `json:"description,omitempty"`
	Sprint               []SprintInfo     `json:"sprint,omitempty"`
	Type                 *IssueType       `json:"type"`
	Priority             *PriorityInfo    `json:"priority"`
	CreatedAt            Datetime         `json:"createdAt"`
	Followers            []User           `json:"followers,omitempty"`
	CreatedBy            *User            `json:"createdBy"`
	Votes                int              `json:"votes"`
	Assignee             *User            `json:"assignee,omitempty"`
	Project              *ProjectDetails  `json:"project,omitempty"`
	Queue                *QueueInfo       `json:"queue"`
	UpdatedAt            Datetime         `json:"updatedAt"`
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

func (t *YandexTrackerClient) doWithRetry(req *http.Request, target any) error {
	var err error
	for range 3 {
		err = t.do(req, target)
		if err == nil {
			return nil
		}
	}
	return err
}

func (t *YandexTrackerClient) do(req *http.Request, target any) error {
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
	req, err := t.newRequest(ctx, "GET", "/v3/issues/_count", nil, nil)
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
	req, err := t.newRequest(ctx, "GET", "/v3/issues", params, nil)
	if err != nil {
		return nil, err
	}

	var issues []*Issue
	if err := t.doWithRetry(req, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

func (t *YandexTrackerClient) ListIssueComments(ctx context.Context, issueKey string, from int, perPage int) ([]*Comment, error) {
	params := url.Values{}
	params.Add("from", fmt.Sprintf("%d", from))
	params.Add("perPage", fmt.Sprintf("%d", perPage))
	params.Add("expand", "attachments")
	req, err := t.newRequest(ctx, "GET", fmt.Sprintf("/v3/issues/%s/comments", issueKey), params, nil)
	if err != nil {
		return nil, err
	}

	var comments []*Comment
	if err := t.doWithRetry(req, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

func (t *YandexTrackerClient) GetAttachment(ctx context.Context, issueKey string, id string) (*Attachment, error) {
	req, err := t.newRequest(ctx, "GET", fmt.Sprintf("/v3/issues/%s/attachments/%s", issueKey, id), nil, nil)
	if err != nil {
		return nil, err
	}

	var attachment Attachment
	if err := t.doWithRetry(req, &attachment); err != nil {
		return nil, err
	}
	return &attachment, nil
}

func (t *YandexTrackerClient) DownloadAttachment(ctx context.Context, issueKey string, id string, name string) ([]byte, error) {
	req, err := t.newRequest(ctx, "GET", fmt.Sprintf("/v3/issues/%s/attachments/%s/%s", issueKey, id, name), nil, nil)
	if err != nil {
		return nil, err
	}

	var content []byte
	if err := t.doWithRetry(req, &content); err != nil {
		return nil, err
	}
	return content, nil
}
