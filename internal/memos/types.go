package memos

import "time"

type SearchRequest struct {
	Query       string
	CreatorID   *int64
	Tag         string
	Visibility  string
	Pinned      *bool
	Limit       int
	Offset      int
	PageToken   string
	OrderBy     string
	ShowDeleted bool
}

type SearchResponse struct {
	Memos         []Memo `json:"memos"`
	NextPageToken string `json:"nextPageToken"`
}

type CreateMemoRequest struct {
    Content     string
    Visibility  string
    Pinned      *bool
    CreateTime  *time.Time
    UpdateTime  *time.Time
}

type UpdateMemoRequest struct {
	Content    *string
	Visibility *string
	Pinned     *bool
}

type Memo struct {
	Name        string   `json:"name"`
	UID         string   `json:"uid"`
	Creator     string   `json:"creator"`
	Content     string   `json:"content"`
	Visibility  string   `json:"visibility"`
	Pinned      bool     `json:"pinned"`
	Tags        []string `json:"tags"`
	CreateTime  string   `json:"createTime"`
	UpdateTime  string   `json:"updateTime"`
	DisplayTime string   `json:"displayTime"`
	Snippet     string   `json:"snippet"`
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return "memos API error"
	}
	return e.Body
}
