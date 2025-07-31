# WhatsApp Bridge

![Go](https://img.shields.io/badge/Go-1.19%2B-blue?style=for-the-badge&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-Database-lightgrey?style=for-the-badge&logo=sqlite&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Database-336791?style=for-the-badge&logo=postgresql&logoColor=white)
![REST API](https://img.shields.io/badge/REST-API-green?style=for-the-badge&logo=fastapi&logoColor=white)
![WhatsApp](https://img.shields.io/badge/WhatsApp-Integration-25D366?style=for-the-badge&logo=whatsapp&logoColor=white)
![MIT License](https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge)

A production-ready WhatsApp bridge built with Go that provides a REST API interface for WhatsApp messaging using the `whatsmeow` library. This bridge allows you to send and receive WhatsApp messages programmatically while maintaining message history in a local SQLite database or a PostgreSQL database.

## Features

- **WhatsApp Integration**: Connect to WhatsApp using QR code authentication
- **Web QR Interface**: Modern web interface for QR code scanning (no terminal access required)
- **REST API**: Send messages, download media, and retrieve chat/message history via HTTP endpoints
- **Message History**: Store and retrieve message history with SQLite or PostgreSQL
- **Media Support**: Send and receive images, videos, audio, and documents
- **Group Chat Support**: Handle both individual and group conversations
- **History Sync**: Sync message history from WhatsApp servers
- **Real-time Messaging**: Receive messages in real-time
- **Database Flexibility**: Supports both SQLite (default) and PostgreSQL (via configuration)

## Prerequisites

- Go 1.19 or higher
- SQLite3 (default) or PostgreSQL (optional)

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
go run main.go database.go qr_web.go
```

The application will:
1. Start the WhatsApp client
2. Launch the web QR interface
3. Display a QR code both in the web interface and terminal (backup)
4. Start the REST API server on port `8080`
5. Begin listening for incoming messages

### QR Code Authentication

#### Web Interface (Recommended)
1. Open your browser and navigate to `http://localhost:8080`
2. You'll see a modern web interface with the QR code
3. Open WhatsApp on your phone
4. Go to **Settings → Linked Devices → Link a Device**
5. Scan the QR code from the web page
6. The page will automatically update when connected

#### Terminal (Backup)
If you prefer the terminal, the QR code is also displayed there as a backup option.

### First Time Setup

1. Run the application
2. Scan the QR code with your WhatsApp mobile app (**WhatsApp > Settings > Linked Devices > Link a Device**)
3. Once connected, the bridge will start receiving messages

## Ports and Services

The WhatsApp Bridge runs all services on a single port:

- **Combined Service**: `http://localhost:8080`
  - Modern web interface for QR code authentication
  - REST API for sending messages, downloading media, etc.
  - Health check endpoint
  - Real-time connection status

This consolidated approach makes the application ideal for deployment on platforms like Google Cloud Run that require a single port.

## API Endpoints

### Send Message

**POST** `/api/send`

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

**POST** `/api/download`

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

**GET** `/api/messages/<chat_jid>?limit=<limit>`

Retrieve message history for a specific chat.

**Parameters:**
- `chat_jid`: WhatsApp JID of the chat
- `limit`: Number of messages to retrieve (optional, default: 50)

### Get Chats

**GET** `/api/chats`

Retrieve a list of all chats with their last message timestamps.

### Database Status

**GET** `/api/db/status`

Check the health and connection status of the database.

## Project Structure

```
whatsapp-bridge/
├── main.go         # Main application code
├── qr_web.go       # QR web interface
├── database.go     # Database adapter
├── Dockerfile      # Docker container definition
└── store/          # Local storage directory
```

## Docker Deployment

### Building the Docker Image

To build the Docker image:

```bash
cd whatsapp-bridge
docker build -t whatsapp-bridge .
```

### Running the Docker Container

To run the Docker container:

```bash
docker run -p 8080:8080 -v $(pwd)/store:/app/store whatsapp-bridge
```

This will:
- Map port 8080 from the container to your host machine
- Mount the local `store` directory to persist data between container restarts

### Environment Variables

The Docker container supports the following environment variables:

- `PORT`: The port to run the server on (default: 8080)
- `DATABASE_URL`: PostgreSQL connection string (optional, falls back to SQLite if not provided)

## Google Cloud Run Deployment

To deploy to Google Cloud Run:

1. Build and push the Docker image to Google Container Registry:

```bash
# Set your Google Cloud project ID
PROJECT_ID=your-project-id

# Build the image with Google Cloud Build
gcloud builds submit --tag gcr.io/$PROJECT_ID/whatsapp-bridge

# Or build locally and push
docker build -t gcr.io/$PROJECT_ID/whatsapp-bridge .
docker push gcr.io/$PROJECT_ID/whatsapp-bridge
```

2. Deploy to Cloud Run:

```bash
gcloud run deploy whatsapp-bridge \
  --image gcr.io/$PROJECT_ID/whatsapp-bridge \
  --platform managed \
  --allow-unauthenticated \
  --region us-central1 \
  --memory 512Mi
```

3. For persistence, consider:
   - Using a PostgreSQL database (set `DATABASE_URL` environment variable)
   - Mounting a persistent volume (for production use)

### Important Cloud Run Considerations

1. **Session Persistence**: WhatsApp sessions need to persist between container restarts. Use PostgreSQL for session storage in production.

2. **QR Code Access**: When deploying to Cloud Run, you'll access the QR code via the deployed URL.

3. **Timeouts**: Configure Cloud Run with appropriate request timeout settings (recommended: 5-10 minutes) to handle long-running operations.

4. **Memory**: Allocate at least 512MB of memory to ensure stable operation.

## License

MIT License