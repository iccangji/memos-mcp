# memos-mcp

MCP (Model Context Protocol) server for the self-hosted notes app [Memos](https://github.com/usememos/memos).

## Features

- Search memos with filters and pagination
- Create, read, update, delete memos
- Access token authentication via Bearer token
- Thin TypeScript wrapper for `npx` stdio startup

## Requirements

- A running Memos instance (default `http://localhost:5230`)
- An access token (Settings → Access Tokens)

## Configuration

Environment variables:

- `MEMOS_BASE_URL` (default: `http://localhost:5230`)
- `MEMOS_ACCESS_TOKEN` or `MEMOS_API_TOKEN`

CLI flags (Go binary):

- `--base-url`
- `--access-token` / `--api-token`
- `--timeout` (seconds)

## MCP Tools

- `memos_search`
  - `query` (string)
  - `creator_id` (number)
  - `tag` (string)
  - `visibility` (PUBLIC/PROTECTED/PRIVATE)
  - `pinned` (boolean)
  - `limit` (number, default 10)
  - `offset` (number, default 0)
  - `page_token` (string)
  - `order_by` (string)
  - `show_deleted` (boolean)

- `memos_get`
  - `memo_uid` (string)

- `memos_create`
  - `content` (string)
  - `visibility` (string, default PRIVATE)
  - `pinned` (boolean)
  - `createTime` (string, optional, RFC3339 format)
  - `updateTime` (string, optional, RFC3339 format)

- `memos_update`
  - `memo_uid` (string)
  - `content` (string)
  - `visibility` (string)
  - `pinned` (boolean)

- `memos_delete`
  - `memo_uid` (string)
  - `force` (boolean)

## Run locally

```bash
go run ./cmd/memos-mcp --base-url http://localhost:5230 --access-token YOUR_TOKEN
```

## MCP client config

```json
{
  "mcpServers": {
    "memos": {
      "command": "memos-mcp",
      "args": ["--base-url", "http://localhost:5230", "--access-token", "YOUR_TOKEN"]
    }
  }
}
```

## NPM wrapper (npx)

```json
{
  "mcpServers": {
    "memos": {
      "command": "npx",
      "args": ["@jtsang/memos-mcp", "--base-url", "http://localhost:5230", "--access-token", "YOUR_TOKEN"]
    }
  }
}
```

## Development

```bash
# Go tests
go test ./...

# JS wrapper tests
cd js
npm install
npm test
```
