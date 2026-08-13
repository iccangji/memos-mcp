package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpgo "github.com/mark3labs/mcp-go/server"

	"github.com/jtsang4/memos-mcp/internal/memos"
)

type Server struct {
	client *memos.Client
}

func NewServer(client *memos.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Register(mcpServer *mcpgo.MCPServer) {
	mcpServer.AddTool(newSearchTool(), s.handleSearch)
	mcpServer.AddTool(newGetTool(), s.handleGet)
	mcpServer.AddTool(newCreateTool(), s.handleCreate)
	mcpServer.AddTool(newUpdateTool(), s.handleUpdate)
	mcpServer.AddTool(newDeleteTool(), s.handleDelete)
}

type memoSummary struct {
	Name        string   `json:"name,omitempty"`
	UID         string   `json:"uid,omitempty"`
	Creator     string   `json:"creator,omitempty"`
	Content     string   `json:"content,omitempty"`
	Visibility  string   `json:"visibility,omitempty"`
	Pinned      bool     `json:"pinned"`
	Tags        []string `json:"tags,omitempty"`
	CreateTime  string   `json:"createTime,omitempty"`
	UpdateTime  string   `json:"updateTime,omitempty"`
	DisplayTime string   `json:"displayTime,omitempty"`
	Snippet     string   `json:"snippet,omitempty"`
}

type searchResult struct {
	Count         int           `json:"count"`
	Memos         []memoSummary `json:"memos"`
	NextPageToken string        `json:"nextPageToken,omitempty"`
}

type memoResult struct {
	Success bool        `json:"success"`
	Memo    memoSummary `json:"memo"`
}

type deleteResult struct {
	Success bool   `json:"success"`
	UID     string `json:"uid"`
	Force   bool   `json:"force"`
}

func newSearchTool() mcp.Tool {
	return mcp.NewTool("memos_search",
		mcp.WithDescription("Search memos with filters and pagination"),
		mcp.WithString("query", mcp.Description("Text to search for in memo content")),
		mcp.WithNumber("creator_id", mcp.Description("Filter by creator user ID")),
		mcp.WithString("tag", mcp.Description("Filter by tag name")),
		mcp.WithString("visibility", mcp.Description("Visibility: PUBLIC, PROTECTED, PRIVATE")),
		mcp.WithBoolean("pinned", mcp.Description("Filter by pinned status")),
		mcp.WithNumber("limit", mcp.Description("Maximum results to return (default 10)")),
		mcp.WithNumber("offset", mcp.Description("Results offset (default 0)")),
		mcp.WithString("page_token", mcp.Description("Page token from a previous response")),
		mcp.WithString("order_by", mcp.Description("Order by fields, e.g. pinned desc, display_time desc")),
		mcp.WithBoolean("show_deleted", mcp.Description("Include deleted memos")),
	)
}

func newGetTool() mcp.Tool {
	return mcp.NewTool("memos_get",
		mcp.WithDescription("Get a memo by UID"),
		mcp.WithString("memo_uid", mcp.Required(), mcp.Description("Memo UID")),
	)
}

func newCreateTool() mcp.Tool {
  return mcp.NewTool("memos_create",
    mcp.WithDescription("Create a new memo"),
    mcp.WithString("content", mcp.Required(), mcp.Description("Memo content in Markdown")),
    mcp.WithString("visibility", mcp.Description("Visibility: PUBLIC, PROTECTED, PRIVATE (default PRIVATE)")),
    mcp.WithBoolean("pinned", mcp.Description("Whether to pin the memo")),
    mcp.WithString("createTime", mcp.Description("Optional create time (nullable)")),
    mcp.WithString("updateTime", mcp.Description("Optional update time (nullable)")),
  )
}

func newUpdateTool() mcp.Tool {
	return mcp.NewTool("memos_update",
		mcp.WithDescription("Update an existing memo"),
		mcp.WithString("memo_uid", mcp.Required(), mcp.Description("Memo UID")),
		mcp.WithString("content", mcp.Description("New memo content")),
		mcp.WithString("visibility", mcp.Description("Visibility: PUBLIC, PROTECTED, PRIVATE")),
		mcp.WithBoolean("pinned", mcp.Description("Whether to pin the memo")),
	)
}

func newDeleteTool() mcp.Tool {
	return mcp.NewTool("memos_delete",
		mcp.WithDescription("Delete a memo by UID"),
		mcp.WithString("memo_uid", mcp.Required(), mcp.Description("Memo UID")),
		mcp.WithBoolean("force", mcp.Description("Force delete even if memo has associated data")),
	)
}

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	query, _ := optionalString(args, "query")
	creatorID, _ := optionalInt64(args, "creator_id")
	tag, _ := optionalString(args, "tag")
	visibility, _ := optionalString(args, "visibility")
	pinned, _ := optionalBool(args, "pinned")
	limit, _ := optionalInt(args, "limit")
	offset, _ := optionalInt(args, "offset")
	pageToken, _ := optionalString(args, "page_token")
	orderBy, _ := optionalString(args, "order_by")
	showDeleted := optionalBoolValue(args, "show_deleted")

	response, err := s.client.SearchMemos(ctx, memos.SearchRequest{
		Query:       query,
		CreatorID:   creatorID,
		Tag:         tag,
		Visibility:  visibility,
		Pinned:      pinned,
		Limit:       limit,
		Offset:      offset,
		PageToken:   pageToken,
		OrderBy:     orderBy,
		ShowDeleted: showDeleted,
	})
	if err != nil {
		return toolError(err), nil
	}

	result := searchResult{
		Count:         len(response.Memos),
		Memos:         summarizeMemos(response.Memos),
		NextPageToken: response.NextPageToken,
	}
	return jsonToolResult(result)
}

func (s *Server) handleGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uid, err := request.RequireString("memo_uid")
	if err != nil {
		return toolError(err), nil
	}

	memo, err := s.client.GetMemo(ctx, uid)
	if err != nil {
		return toolError(err), nil
	}

	return jsonToolResult(summarizeMemo(*memo))
}

func (s *Server) handleCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := request.RequireString("content")
	if err != nil {
		return toolError(err), nil
	}
	visibility, _ := optionalString(request.GetArguments(), "visibility")
	pinned, _ := optionalBool(request.GetArguments(), "pinned")
	createTimeStr, _ := optionalString(request.GetArguments(), "createTime")
	updateTimeStr, _ := optionalString(request.GetArguments(), "updateTime")

	var createTimePtr *time.Time
	if createTimeStr != "" {
		if ct, err := time.Parse(time.RFC3339, createTimeStr); err == nil {
			createTimePtr = &ct
		} else {
			return toolError(fmt.Errorf("invalid createTime format, expected RFC3339: %w", err)), nil
		}
	}
	var updateTimePtr *time.Time
	if updateTimeStr != "" {
		if ut, err := time.Parse(time.RFC3339, updateTimeStr); err == nil {
			updateTimePtr = &ut
		} else {
			return toolError(fmt.Errorf("invalid updateTime format, expected RFC3339: %w", err)), nil
		}
	}

	memo, err := s.client.CreateMemo(ctx, memos.CreateMemoRequest{
		Content:     content,
		Visibility:  visibility,
		Pinned:      pinned,
		CreateTime:  createTimePtr,
		UpdateTime:  updateTimePtr,
	})
	if err != nil {
		return toolError(err), nil
	}

	return jsonToolResult(memoResult{
		Success: true,
		Memo:    summarizeMemo(*memo),
	})
}

func (s *Server) handleUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uid, err := request.RequireString("memo_uid")
	if err != nil {
		return toolError(err), nil
	}
	args := request.GetArguments()
	content, _ := optionalString(args, "content")
	visibility, _ := optionalString(args, "visibility")
	pinned, _ := optionalBool(args, "pinned")

	var contentPtr *string
	if content != "" {
		contentPtr = &content
	}
	var visibilityPtr *string
	if visibility != "" {
		visibilityPtr = &visibility
	}

	memo, err := s.client.UpdateMemo(ctx, uid, memos.UpdateMemoRequest{
		Content:    contentPtr,
		Visibility: visibilityPtr,
		Pinned:     pinned,
	})
	if err != nil {
		return toolError(err), nil
	}

	return jsonToolResult(memoResult{
		Success: true,
		Memo:    summarizeMemo(*memo),
	})
}

func (s *Server) handleDelete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	uid, err := request.RequireString("memo_uid")
	if err != nil {
		return toolError(err), nil
	}
	force, _ := optionalBool(request.GetArguments(), "force")
	forceValue := false
	if force != nil {
		forceValue = *force
	}

	if err := s.client.DeleteMemo(ctx, uid, forceValue); err != nil {
		return toolError(err), nil
	}

	return jsonToolResult(deleteResult{
		Success: true,
		UID:     uid,
		Force:   forceValue,
	})
}

func summarizeMemos(memosList []memos.Memo) []memoSummary {
	if len(memosList) == 0 {
		return nil
	}
	result := make([]memoSummary, 0, len(memosList))
	for _, memo := range memosList {
		result = append(result, summarizeMemo(memo))
	}
	return result
}

func summarizeMemo(memo memos.Memo) memoSummary {
	return memoSummary{
		Name:        memo.Name,
		UID:         memo.UID,
		Creator:     memo.Creator,
		Content:     memo.Content,
		Visibility:  memo.Visibility,
		Pinned:      memo.Pinned,
		Tags:        memo.Tags,
		CreateTime:  memo.CreateTime,
		UpdateTime:  memo.UpdateTime,
		DisplayTime: memo.DisplayTime,
		Snippet:     memo.Snippet,
	}
}

func jsonToolResult(value any) (*mcp.CallToolResult, error) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return toolError(fmt.Errorf("encode response: %w", err)), nil
	}
	return mcp.NewToolResultText(string(payload)), nil
}

func toolError(err error) *mcp.CallToolResult {
	return mcp.NewToolResultError(err.Error())
}
