# WhatsApp Bridge

A production-ready WhatsApp bridge built with Go that provides a REST API interface for WhatsApp messaging using the whatsmeow library. This bridge allows you to send and receive WhatsApp messages programmatically while maintaining message history in a local SQLite database.

## Features

- **WhatsApp Integration**: Connect to WhatsApp using QR code authentication
- **Web QR Interface**: Modern web interface for QR code scanning (no terminal access required)
- **REST API**: Send messages and download media via HTTP endpoints
- **Message History**: Store and retrieve message history with SQLite database
- **Media Support**: Send and receive images, videos, audio, and documents
- **Group Chat Support**: Handle both individual and group conversations
- **History Sync**: Sync message history from WhatsApp servers
- **Real-time Messaging**: Receive messages in real-time

## Prerequisites

- Go 1.19 or higher
- SQLite3

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd whatsapp-bridge
```

2. Install dependencies:
```bash
cd whatsapp-bridge
go mod tidy
```

## Usage

### Running the Application

From the project root directory:

```bash
cd whatsapp-bridge
go run main.go qr_web.go
```

The application will:
1. Start the WhatsApp client
2. Launch the web QR interface on port 3000
3. Display a QR code both in the web interface and terminal (backup)
4. Start the REST API server on port 8080
5. Begin listening for incoming messages

### QR Code Authentication

#### Web Interface (Recommended)
1. Open your browser and navigate to `http://localhost:3000`
2. You'll see a modern web interface with the QR code
3. Open WhatsApp on your phone
4. Go to Settings → Linked Devices → Link a Device
5. Scan the QR code from the web page
6. The page will automatically update when connected

#### Terminal (Backup)
If you prefer the terminal, the QR code is also displayed there as a backup option.

### First Time Setup

1. Run the application
2. Scan the QR code with your WhatsApp mobile app (WhatsApp > Settings > Linked Devices > Link a Device)
3. Once connected, the bridge will start receiving messages

## Ports and Services

The WhatsApp Bridge runs two services simultaneously:

- **Web QR Interface**: `http://localhost:3000`
  - Modern web interface for QR code authentication
  - Auto-refreshing status updates
  - Mobile-friendly responsive design
  - Real-time connection status

- **REST API Server**: `http://localhost:8080`
  - Send messages via HTTP POST requests
  - Download media files
  - Retrieve message history
  - All programmatic WhatsApp operations

## API Endpoints

### Send Message

**POST** `/send`

Send a text message or media file to a WhatsApp contact or group.

**Request Body:**
```json
{
  "recipient": "1234567890@s.whatsapp.net",
  "message": "Hello, World!",
  "media_path": "/path/to/file.jpg" // Optional for media
}
```

**Response:**
```json
{
  "success": true,
  "message": "Message sent successfully"
}
```

### Download Media

**POST** `/download`

Download media from a received message.

**Request Body:**
```json
{
  "message_id": "message_id_here",
  "chat_jid": "1234567890@s.whatsapp.net"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Media downloaded successfully",
  "filename": "downloaded_file.jpg",
  "path": "/path/to/downloaded/file.jpg"
}
```

### Get Messages

**GET** `/messages?chat_jid=<chat_jid>&limit=<limit>`

Retrieve message history for a specific chat.

**Parameters:**
- `chat_jid`: WhatsApp JID of the chat
- `limit`: Number of messages to retrieve (optional, default: 50)

### Get Chats

**GET** `/chats`

Retrieve list of all chats with their last message timestamps.

## Project Structure

```
whatsapp-bridge/
├── whatsapp-bridge/
│   ├── main.go          # Main application file
│   ├── go.mod           # Go module dependencies
│   └── go.sum           # Go module checksums
├── store/               # Created at runtime
│   ├── messages.db      # SQLite database for messages
│   └── device.db        # WhatsApp session storage
├── downloads/           # Created at runtime for media downloads
└── README.md
```

## Database Schema

The application creates two main tables:

### Chats Table
- `jid`: Chat identifier (PRIMARY KEY)
- `name`: Chat name
- `last_message_time`: Timestamp of last message

### Messages Table
- `id`: Message ID
- `chat_jid`: Chat identifier (FOREIGN KEY)
- `sender`: Message sender
- `content`: Message content
- `timestamp`: Message timestamp
- `is_from_me`: Boolean indicating if message was sent by this client
- `media_type`: Type of media (if any)
- `filename`: Original filename (for media)
- Additional media metadata fields

## Configuration

The application uses the following default settings:
- **API Port**: 8080
- **Database Path**: `store/messages.db`
- **Downloads Path**: `downloads/`
- **Session Storage**: `store/device.db`

## Troubleshooting

### Common Issues

1. **"no such file or directory" error**: Make sure you're running the command from the correct directory:
   ```bash
   cd whatsapp-bridge
   go run main.go
   ```

2. **QR Code not displaying**: Ensure your terminal supports UTF-8 characters

3. **Database errors**: Check that the `store/` directory is writable

4. **Connection issues**: Verify your internet connection and WhatsApp account status

### Logs

The application provides detailed logging for:
- Message sending/receiving
- Database operations
- API requests
- Connection status

## Dependencies

Key dependencies include:
- `go.mau.fi/whatsmeow`: WhatsApp Web API library
- `github.com/mattn/go-sqlite3`: SQLite driver
- `github.com/mdp/qrterminal`: QR code terminal display

## License

MIT License

## Contributing

PRs welcome!