package main

import (
	"bytes"
	"fmt"
	"image/png"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/skip2/go-qrcode"
	"github.com/supabase-community/supabase-go"
)

// QRWebServer handles serving QR codes via web interface
type QRWebServer struct {
	currentQRCode string
	qrMutex       sync.RWMutex
	isConnected   bool
	supabaseClient *supabase.Client
	supabaseURL    string
	supabaseKey    string
}

// NewQRWebServer creates a new QR web server instance
func NewQRWebServer() *QRWebServer {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_ANON_KEY")
	
	var client *supabase.Client
	if supabaseURL != "" && supabaseKey != "" {
		var err error
		client, err = supabase.NewClient(supabaseURL, supabaseKey, &supabase.ClientOptions{})
		if err != nil {
			fmt.Printf("Failed to initialize Supabase client: %v\n", err)
		}
	}
	
	return &QRWebServer{
		supabaseClient: client,
		supabaseURL:    supabaseURL,
		supabaseKey:    supabaseKey,
	}
}

// UpdateQRCode updates the current QR code
func (q *QRWebServer) UpdateQRCode(code string) {
	q.qrMutex.Lock()
	defer q.qrMutex.Unlock()
	q.currentQRCode = code
	q.isConnected = false
}

// SetConnected marks the connection as successful
func (q *QRWebServer) SetConnected() {
	q.qrMutex.Lock()
	defer q.qrMutex.Unlock()
	q.isConnected = true
	q.currentQRCode = ""
}

// GetQRCode returns the current QR code
func (q *QRWebServer) GetQRCode() (string, bool) {
	q.qrMutex.RLock()
	defer q.qrMutex.RUnlock()
	return q.currentQRCode, q.isConnected
}

// getSessionFromRequest extracts session token from request (cookie or Authorization header)
func (q *QRWebServer) getSessionFromRequest(r *http.Request) string {
	// First try Authorization header
	auth := r.Header.Get("Authorization")
	if auth != "" && strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	
	// Then try cookie
	cookie, err := r.Cookie("sb-access-token")
	if err == nil {
		return cookie.Value
	}
	
	return ""
}

// validateSession validates a Supabase session token
func (q *QRWebServer) validateSession(sessionToken string) bool {
	if sessionToken == "" || q.supabaseClient == nil {
		return false
	}
	
	// Use Supabase client to validate the session
	// For now, we'll do a simple check - in production you'd validate with Supabase
	// This is a placeholder that assumes any non-empty token is valid
	// You can enhance this by calling Supabase's user endpoint
	return len(sessionToken) > 10 // Basic validation
}

// authMiddleware wraps HTTP handlers with authentication
func (q *QRWebServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if no Supabase client is configured (development mode)
		if q.supabaseClient == nil {
			next(w, r)
			return
		}
		
		sessionToken := q.getSessionFromRequest(r)
		if !q.validateSession(sessionToken) {
			// Redirect to login page
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		
		next(w, r)
	}
}

// ServeQRPage serves the main QR code page or dashboard
func (q *QRWebServer) ServeQRPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>WhatsApp Bridge</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #25D366 0%, #128C7E 100%);
            margin: 0;
            padding: 20px;
            min-height: 100vh;
        }
        .container {
            background: white;
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
            max-width: 800px;
            width: 100%;
            margin: 0 auto;
        }
        .qr-container {
            max-width: 500px;
            margin: 0 auto;
        }
        .dashboard {
            text-align: left;
            max-width: 100%;
        }
        .logo {
            font-size: 2.5em;
            color: #25D366;
            margin-bottom: 10px;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 1.8em;
        }
        .subtitle {
            color: #666;
            margin-bottom: 30px;
            font-size: 1.1em;
        }
        .qr-code-area {
            background: #f8f9fa;
            border-radius: 15px;
            padding: 30px;
            margin: 30px 0;
            border: 2px dashed #ddd;
        }
        .qr-code {
            max-width: 100%;
            height: auto;
            border-radius: 10px;
        }
        .status {
            padding: 15px;
            border-radius: 10px;
            margin: 20px 0;
            font-weight: 500;
        }
        .status.waiting {
            background: #fff3cd;
            color: #856404;
            border: 1px solid #ffeaa7;
        }
        .status.connected {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .status.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        .refresh-btn {
            background: #25D366;
            color: white;
            border: none;
            padding: 12px 24px;
            border-radius: 25px;
            cursor: pointer;
            font-size: 1em;
            font-weight: 500;
            transition: background 0.3s;
            margin: 10px 5px;
        }
        .refresh-btn:hover {
            background: #128C7E;
        }
        .instructions {
            background: #e3f2fd;
            padding: 20px;
            border-radius: 10px;
            margin: 20px 0;
            text-align: left;
        }
        .instructions ol {
            margin: 0;
            padding-left: 20px;
        }
        .instructions li {
            margin: 8px 0;
            color: #1565c0;
        }
        .dashboard-section {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 20px;
            margin: 20px 0;
        }
        .dashboard-section h3 {
            margin-top: 0;
            color: #333;
        }
        .message-list {
            max-height: 300px;
            overflow-y: auto;
            border: 1px solid #ddd;
            border-radius: 8px;
            padding: 10px;
            background: white;
        }
        .message-item {
            padding: 10px;
            border-bottom: 1px solid #eee;
            margin-bottom: 10px;
        }
        .message-item:last-child {
            border-bottom: none;
            margin-bottom: 0;
        }
        .message-sender {
            font-weight: bold;
            color: #25D366;
        }
        .message-time {
            font-size: 0.8em;
            color: #666;
        }
        .message-content {
            margin-top: 5px;
        }
        .send-message-form {
            background: white;
            padding: 20px;
            border-radius: 8px;
            border: 1px solid #ddd;
        }
        .form-group {
            margin-bottom: 15px;
        }
        .form-group label {
            display: block;
            margin-bottom: 5px;
            font-weight: 500;
        }
        .form-group input, .form-group textarea {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 14px;
            box-sizing: border-box;
        }
        .form-group textarea {
            height: 80px;
            resize: vertical;
        }
        .send-btn {
            background: #25D366;
            color: white;
            border: none;
            padding: 12px 30px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 1em;
            font-weight: 500;
        }
        .send-btn:hover {
            background: #128C7E;
        }
        .send-btn:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        .loading {
            text-align: center;
            color: #666;
            padding: 20px;
        }
        .error {
            color: #dc3545;
            background: #f8d7da;
            padding: 10px;
            border-radius: 5px;
            margin: 10px 0;
        }
        .success {
            color: #155724;
            background: #d4edda;
            padding: 10px;
            border-radius: 5px;
            margin: 10px 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">📱</div>
        <h1>WhatsApp Bridge</h1>
        
        <div id="content">
            <div class="loading">Loading...</div>
        </div>
    </div>
    
    <script>
        let isConnected = false;
        let refreshInterval;
        
        function showQRInterface() {
            return '<div class="qr-container">' +
                   '<p class="subtitle">Scan QR Code to Connect</p>' +
                   '<div id="qr-status"></div>' +
                   '<div class="instructions">' +
                   '<strong>How to connect:</strong>' +
                   '<ol>' +
                   '<li>Open WhatsApp on your phone</li>' +
                   '<li>Go to Settings &rarr; Linked Devices</li>' +
                   '<li>Tap "Link a Device"</li>' +
                   '<li>Scan the QR code above</li>' +
                   '</ol>' +
                   '</div>' +
                   '<button class="refresh-btn" onclick="refreshStatus()">Refresh</button>' +
                   '</div>';
        }
        
        function showDashboard() {
            return '<div class="dashboard">' +
                   '<div class="status connected">&#x2705; Connected to WhatsApp!</div>' +
                   '<div class="dashboard-section">' +
                   '<h3>&#x1F4CB; Recent Messages</h3>' +
                   '<div id="message-list" class="message-list">' +
                   '<div class="loading">Loading messages...</div>' +
                   '</div>' +
                   '<button class="refresh-btn" onclick="loadMessages()">Refresh Messages</button>' +
                   '</div>' +
                   '<div class="dashboard-section">' +
                   '<h3>&#x1F4E4; Send Message</h3>' +
                   '<div class="send-message-form">' +
                   '<div class="form-group">' +
                   '<label for="recipient">Recipient Phone Number:</label>' +
                   '<input type="text" id="recipient" placeholder="e.g., +1234567890" />' +
                   '</div>' +
                   '<div class="form-group">' +
                   '<label for="message">Message:</label>' +
                   '<textarea id="message" placeholder="Type your message here..."></textarea>' +
                   '</div>' +
                   '<button class="send-btn" onclick="sendMessage()" id="send-btn">Send Message</button>' +
                   '<div id="send-result"></div>' +
                   '</div>' +
                   '</div>' +
                   '</div>';
        }
        
        function refreshStatus() {
            fetch('/qr/status')
                .then(response => response.json())
                .then(data => {
                    const content = document.getElementById('content');
                    
                    if (data.connected) {
                        if (!isConnected) {
                            isConnected = true;
                            content.innerHTML = showDashboard();
                            loadMessages();
                            // Stop auto-refresh when connected
                            if (refreshInterval) {
                                clearInterval(refreshInterval);
                            }
                        }
                    } else {
                        if (isConnected) {
                            isConnected = false;
                            content.innerHTML = showQRInterface();
                            // Restart auto-refresh
                            startAutoRefresh();
                        } else if (!document.getElementById('qr-status')) {
                            // This handles the initial load when the QR interface isn't yet visible.
                            content.innerHTML = showQRInterface();
                        }
                        updateQRStatus(data);
                    }
                })
                .catch(err => {
                    console.error('Error fetching status:', err);
                    const content = document.getElementById('content');
                    // Avoid being stuck on "Loading..." if the server is unreachable.
                    if (!document.getElementById('qr-status')) {
                        content.innerHTML = showQRInterface();
                    }
                    const qrStatus = document.getElementById('qr-status');
                    if (qrStatus) {
                        qrStatus.innerHTML = '<div class="status error">Could not connect to the server. Retrying...</div>';
                    }
                });
        }
        
        function updateQRStatus(data) {
            const qrStatus = document.getElementById('qr-status');
            if (!qrStatus) return;
            
            if (data.qr_available) {
                qrStatus.innerHTML = '<div class="status waiting">&#x23F3; Waiting for QR code scan...</div>' +
                                   '<div class="qr-code-area">' +
                                   '<img src="/qr/image" alt="QR Code" class="qr-code" />' +
                                   '</div>';
            } else {
                qrStatus.innerHTML = '<div class="status waiting">&#x23F3; Generating QR code...</div>';
            }
        }
        
        function loadMessages() {
            const messageList = document.getElementById('message-list');
            if (!messageList) return;
            
            messageList.innerHTML = '<div class="loading">Loading messages...</div>';
            
            // Get list of chats first
            fetch('/api/chats')
                .then(response => response.json())
                .then(chats => {
                    if (chats && Object.keys(chats).length > 0) {
                        // Get the first chat's messages as a sample
                        const firstChatJID = Object.keys(chats)[0];
                        return fetch('/api/messages/' + encodeURIComponent(firstChatJID) + '?limit=10');
                    } else {
                        throw new Error('No chats found');
                    }
                })
                .then(response => response.json())
                .then(messages => {
                    if (messages && messages.length > 0) {
                        let html = '';
                        messages.forEach(msg => {
                            const time = new Date(msg.time).toLocaleString();
                            html += '<div class="message-item">' +
                                   '<div class="message-sender">' + (msg.Sender || 'Unknown') + '</div>' +
                                   '<div class="message-time">' + msg.Time + '</div>' +
                                   '<div class="message-content">' + (msg.Content || '[Media]') + '</div>' +
                                   '</div>';
                        });
                        messageList.innerHTML = html;
                    } else {
                        messageList.innerHTML = '<div class="loading">No messages found. Try sending a message to see it here.</div>';
                    }
                })
                .catch(err => {
                    console.error('Error loading messages:', err);
                    messageList.innerHTML = '<div class="error">Failed to load messages. Make sure the API is running.</div>';
                });
        }
        
        function sendMessage() {
            const recipient = document.getElementById('recipient').value.trim();
            const message = document.getElementById('message').value.trim();
            const sendBtn = document.getElementById('send-btn');
            const resultDiv = document.getElementById('send-result');
            
            if (!recipient || !message) {
                resultDiv.innerHTML = '<div class="error">Please fill in both recipient and message fields.</div>';
                return;
            }
            
            sendBtn.disabled = true;
            sendBtn.textContent = 'Sending...';
            resultDiv.innerHTML = '';
            
            fetch('/api/send', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    recipient: recipient,
                    message: message
                })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    resultDiv.innerHTML = '<div class="success">&#x2705; Message sent successfully!</div>';
                    document.getElementById('message').value = '';
                    // Refresh messages to show the sent message
                    setTimeout(loadMessages, 1000);
                } else {
                    resultDiv.innerHTML = '<div class="error">&#x274C; Failed to send message: ' + data.message + '</div>';
                }
            })
            .catch(err => {
                console.error('Error sending message:', err);
                resultDiv.innerHTML = '<div class="error">&#x274C; Network error. Make sure the API is running.</div>';
            })
            .finally(() => {
                sendBtn.disabled = false;
                sendBtn.textContent = 'Send Message';
            });
        }
        
        function startAutoRefresh() {
            if (refreshInterval) {
                clearInterval(refreshInterval);
            }
            refreshInterval = setInterval(refreshStatus, 3000);
        }
        
        // Initialize
        document.addEventListener('DOMContentLoaded', function() {
            refreshStatus();
            startAutoRefresh();
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
}

// ServeLoginPage serves the login page with Supabase Auth
func (q *QRWebServer) ServeLoginPage(w http.ResponseWriter, r *http.Request) {
	// Handle POST request for login
	if r.Method == "POST" {
		q.handleLogin(w, r)
		return
	}
	
	// If already authenticated, redirect to main page
	sessionToken := q.getSessionFromRequest(r)
	if q.validateSession(sessionToken) {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
		loginTmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Login - WhatsApp Bridge</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #25D366 0%, #128C7E 100%);
            margin: 0;
            padding: 20px;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            background: white;
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
            max-width: 400px;
            width: 100%;
        }
        .logo {
            font-size: 3em;
            color: #25D366;
            margin-bottom: 10px;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 1.8em;
        }
        .subtitle {
            color: #666;
            margin-bottom: 30px;
            font-size: 1.1em;
        }
        .form-group {
            margin: 15px 0;
            text-align: left;
        }
        .form-group label {
            display: block;
            margin-bottom: 5px;
            color: #333;
            font-weight: 500;
        }
        .form-group input {
            width: 100%;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 1em;
            box-sizing: border-box;
        }
        .login-btn {
            background: #25D366;
            color: white;
            border: none;
            padding: 12px 30px;
            border-radius: 25px;
            cursor: pointer;
            font-size: 1em;
            font-weight: 500;
            width: 100%;
            margin: 20px 0;
        }
        .login-btn:hover {
            background: #128C7E;
        }
        .login-btn:disabled {
            background: #ccc;
            cursor: not-allowed;
        }
        .error {
            background: #f8d7da;
            color: #721c24;
            padding: 10px;
            border-radius: 5px;
            margin: 10px 0;
            border: 1px solid #f5c6cb;
        }
        .success {
            background: #d4edda;
            color: #155724;
            padding: 10px;
            border-radius: 5px;
            margin: 10px 0;
            border: 1px solid #c3e6cb;
        }
        .info {
            background: #d1ecf1;
            color: #0c5460;
            padding: 10px;
            border-radius: 5px;
            margin: 10px 0;
            border: 1px solid #bee5eb;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="logo">📱</div>
        <h1>WhatsApp Bridge</h1>
        <p class="subtitle">Please log in to access the QR code interface</p>
        
        <div id="message"></div>
        
        <form method="POST" action="/login">
            <div class="form-group">
                <label for="email">Email:</label>
                <input type="email" id="email" name="email" required>
            </div>
            <div class="form-group">
                <label for="password">Password:</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="login-btn">Login</button>
        </form>
        
        <div class="info">
            <small>Development mode: Authentication is ` + func() string {
				if q.supabaseClient == nil {
					return "disabled"
				}
				return "enabled"
			}() + `</small>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(loginTmpl))
}

// handleLogin processes the login form submission
func (q *QRWebServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	password := r.FormValue("password")
	
	if email == "" || password == "" {
		http.Redirect(w, r, "/login?error=missing_fields", http.StatusTemporaryRedirect)
		return
	}
	
	// If no Supabase client (development mode), accept any login
	if q.supabaseClient == nil {
		// Set a dummy session cookie for development
		http.SetCookie(w, &http.Cookie{
			Name:     "sb-access-token",
			Value:    "dev-session-token",
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			SameSite: http.SameSiteStrictMode,
		})
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	
	// Use Supabase client to authenticate
	response, err := q.supabaseClient.Auth.SignInWithEmailPassword(email, password)
	if err != nil {
		fmt.Printf("Login error: %v\n", err)
		http.Redirect(w, r, "/login?error=invalid_credentials", http.StatusTemporaryRedirect)
		return
	}
	
	// Set session cookie with the access token
	if response.AccessToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "sb-access-token",
			Value:    response.AccessToken,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			SameSite: http.SameSiteStrictMode,
		})
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "/login?error=no_token", http.StatusTemporaryRedirect)
	}
}

// ServeAuthCallback handles the Supabase auth callback
func (q *QRWebServer) ServeAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Extract access token from URL fragment (handled by JavaScript on login page)
	// This endpoint mainly serves as a landing page for the auth flow
	callbackTmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Authentication - WhatsApp Bridge</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #25D366 0%, #128C7E 100%);
            margin: 0;
            padding: 20px;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .callback-container {
            background: white;
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
            max-width: 400px;
            width: 100%;
        }
        .logo {
            font-size: 3em;
            color: #25D366;
            margin-bottom: 10px;
        }
        .status {
            padding: 15px;
            border-radius: 10px;
            margin: 20px 0;
            font-weight: 500;
        }
        .success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
    </style>
</head>
<body>
    <div class="callback-container">
        <div class="logo">🔐</div>
        <h1>Authentication</h1>
        <div id="status" class="status">Processing authentication...</div>
    </div>

    <script>
        // Extract token from URL fragment
        const hash = window.location.hash.substring(1);
        const params = new URLSearchParams(hash);
        const accessToken = params.get('access_token');
        const error = params.get('error');
        
        if (error) {
            document.getElementById('status').className = 'status error';
            document.getElementById('status').textContent = 'Authentication failed: ' + error;
        } else if (accessToken) {
            // Store token in cookie
            document.cookie = 'sb-access-token=' + accessToken + '; path=/; max-age=3600; secure; samesite=strict';
            document.getElementById('status').className = 'status success';
            document.getElementById('status').textContent = 'Authentication successful! Redirecting...';
            
            // Redirect to main page after a short delay
            setTimeout(() => {
                window.location.href = '/';
            }, 2000);
        } else {
            document.getElementById('status').className = 'status error';
            document.getElementById('status').textContent = 'No authentication token received.';
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(callbackTmpl))
}

// ServeQRImage serves the QR code as a PNG image
func (q *QRWebServer) ServeQRImage(w http.ResponseWriter, r *http.Request) {
	code, connected := q.GetQRCode()
	
	if connected {
		http.Error(w, "Already connected", http.StatusGone)
		return
	}
	
	if code == "" {
		http.Error(w, "No QR code available", http.StatusNotFound)
		return
	}

	// Generate QR code image
	qr, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	// Create PNG image
	img := qr.Image(256)
	
	// Encode to PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		http.Error(w, "Failed to encode QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(buf.Bytes())
}

// ServeQRStatus serves the current QR status as JSON
func (q *QRWebServer) ServeQRStatus(w http.ResponseWriter, r *http.Request) {
	code, connected := q.GetQRCode()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	
	// Simple JSON encoding
	if connected {
		w.Write([]byte(`{"connected": true, "qr_available": false}`))
	} else if code != "" {
		w.Write([]byte(`{"connected": false, "qr_available": true}`))
	} else {
		w.Write([]byte(`{"connected": false, "qr_available": false}`))
	}
}

// RegisterRoutes registers the QR web server routes to the default HTTP mux
func (q *QRWebServer) RegisterRoutes() {
	// Protected routes (require authentication)
	http.HandleFunc("/", q.authMiddleware(q.ServeQRPage))
	http.HandleFunc("/qr/image", q.authMiddleware(q.ServeQRImage))
	http.HandleFunc("/qr/status", q.authMiddleware(q.ServeQRStatus))
	
	// Public routes (no authentication required)
	http.HandleFunc("/login", q.ServeLoginPage)
	http.HandleFunc("/auth/callback", q.ServeAuthCallback)
	
	fmt.Println("QR Web Server routes registered with authentication")
}

// StartQRWebServer starts the QR web server (legacy method, kept for compatibility)
func (q *QRWebServer) StartQRWebServer(port int) {
	// Instead of starting a separate server, just register routes
	q.RegisterRoutes()
	fmt.Printf("QR Web Server routes registered (legacy port %d ignored)\n", port)
}
