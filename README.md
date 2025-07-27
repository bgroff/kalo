# Kalo

A terminal-based API client for testing HTTP requests, built with Go and Bubble Tea. Kalo provides a clean, keyboard-driven interface for managing and executing API requests using the Bruno (.bru) file format.

## Features

- 🚀 **Terminal-based UI** - Fast, keyboard-driven interface
- 📁 **Collection Management** - Organize requests by collections and tags
- 🏷️ **Tag-based Organization** - Group requests with tags for better organization
- 📄 **Bruno File Format** - Compatible with Bruno API client files (.bru)
- 🔍 **JSON Filtering** - Filter JSON responses with jq expressions (Ctrl+J)
- 📊 **Response Visualization** - View headers, body, and status codes
- 🔗 **OpenAPI Import** - Import requests from OpenAPI/Swagger specifications
- ⚡ **Fast Navigation** - Quick switching between requests and collections

## Installation

### Prerequisites

- Go 1.24.5 or later

### Building from Source

```bash
# Clone the repository
git clone <repository-url>
cd kalo

# Install dependencies
make deps

# Build the application
make build

# Run the application
./kalo
```

### Alternative Build Commands

```bash
# Build and run in one command
make run

# Install to /usr/local/bin
sudo make install

# Build for Linux
make build-linux
```

## Usage

### Starting Kalo

```bash
./kalo
```

The application will look for `.bru` files in a `collections/` directory in the current working directory.

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` | Switch between panels (Collections, Request, Response) |
| `↑/↓` | Navigate within panels |
| `Enter` | Execute selected request / Open command palette |
| `Ctrl+C` | Quit application |
| `Ctrl+N` | Create new request |
| `Ctrl+J` | Apply jq filter to JSON responses |
| `q` | Quit application |

### Panel Navigation

**Collections Panel:**
- Navigate through your request collections and individual requests
- Requests are organized by collection folders and tags
- Use `↑/↓` to select different requests

**Request Panel:**
- View the selected request details including:
  - HTTP method and URL
  - Headers
  - Query parameters
  - Request body

**Response Panel:**
- View response after executing a request:
  - Status code and response time
  - Response headers
  - Response body (with JSON pretty-printing)
- Use `↑/↓` to switch between headers and body sections
- Use `Ctrl+J` to filter JSON responses with jq expressions

### File Structure

Kalo expects your API collections to be organized in a `collections/` directory:

```
collections/
├── user-management/
│   ├── create-user.bru
│   ├── get-users.bru
│   └── update-user.bru
└── auth/
    ├── login.bru
    └── refresh-token.bru
```

### Bruno File Format

Kalo uses the Bruno file format (.bru). Here's an example:

```
meta {
  name: Get Users
  type: http
  seq: 1
}

tags {
  users
  api
}

get {
  url: https://api.example.com/users
  body: none
  auth: none
}

headers {
  Accept: application/json
  Content-Type: application/json
}

query {
  limit: 10
  offset: 0
}
```

### Command Palette Features

Press `Enter` in any panel to open the command palette and access:

- **New Request** - Create a new API request file
- **Import from OpenAPI** - Import requests from OpenAPI/Swagger specifications
- **jq Filter** (JSON responses only) - Filter response data with jq expressions

### jq Filtering

When viewing JSON responses, press `Ctrl+J` to open the jq filter dialog. You can use any valid jq expression:

```
# Filter specific fields
.data.users[]

# Get array length
.data | length

# Filter by condition
.data.users[] | select(.active == true)
```

## Examples

The repository includes example `.bru` files in the `examples/` directory to help you get started.

## Development

### Running Tests

```bash
make test
```

### Cleaning Build Artifacts

```bash
make clean
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal UIs
- [gojq](https://github.com/itchyny/gojq) - jq implementation in Go

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

Bryce Groff - bgroff@hawaii.edu