package main

import (
	"bytes"
	"fmt"
	"image/png"
	"net/http"
	"sync"

	"github.com/skip2/go-qrcode"
)

// QRWebServer handles serving QR codes via web interface
type QRWebServer struct {
	currentQRCode string
	qrMutex       sync.RWMutex
	isConnected   bool
}

// NewQRWebServer creates a new QR web server instance
func NewQRWebServer() *QRWebServer {
	return &QRWebServer{}
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

// ServeQRPage serves the main QR code page
func (q *QRWebServer) ServeQRPage(w http.ResponseWriter, r *http.Request) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>WhatsApp Bridge - QR Code Login</title>
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
        .container {
            background: white;
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
            max-width: 500px;
            width: 100%;
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
        .qr-container {
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
        .db-status {
            background: #f8f9fa;
            border: 1px solid #dee2e6;
            border-radius: 8px;
            padding: 15px;
            margin: 20px 0;
            text-align: left;
            font-size: 0.9em;
        }
        .db-status h4 {
            margin: 0 0 10px 0;
            color: #495057;
        }
        .db-indicator {
            display: inline-block;
            width: 10px;
            height: 10px;
            border-radius: 50%;
            margin-right: 8px;
        }
        .db-indicator.healthy {
            background: #28a745;
        }
        .db-indicator.unhealthy {
            background: #dc3545;
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
    </style>
    <script>
        function refreshPage() {
            location.reload();
        }
        
        // Auto-refresh every 3 seconds to check for updates
        setInterval(refreshPage, 3000);
    </script>
</head>
<body>
    <div class="container">
        <div class="logo">📱</div>
        <h1>WhatsApp Bridge</h1>
        <p class="subtitle">Scan QR Code to Connect</p>
        
        <div id="content">
            <!-- Content will be loaded here -->
        </div>
        
        <div class="instructions">
            <strong>How to connect:</strong>
            <ol>
                <li>Open WhatsApp on your phone</li>
                <li>Go to Settings → Linked Devices</li>
                <li>Tap "Link a Device"</li>
                <li>Scan the QR code above</li>
            </ol>
        </div>
        
        <button class="refresh-btn" onclick="refreshPage()">Refresh</button>
    </div>
    
    <script>
        // Function to load database status
        function loadDatabaseStatus() {
            return fetch('http://localhost:8080/api/db/status')
                .then(response => response.json())
                .catch(err => {
                    console.error('Failed to fetch database status:', err);
                    return {
                        healthy: false,
                        status: 'Failed to connect to API',
                        database_info: { type: 'unknown' }
                    };
                });
        }

        // Function to render database status
        function renderDatabaseStatus(dbData) {
            const indicator = dbData.healthy ? 'healthy' : 'unhealthy';
            const dbType = dbData.database_info?.type || 'unknown';
            const driver = dbData.database_info?.driver || 'unknown';
            const modeText = dbData.database_info?.is_remote ? 'Remote (Supabase)' : 'Local Development';
            
            return '<div class="db-status">' +
                   '<h4><span class="db-indicator ' + indicator + '"></span>Database Status</h4>' +
                   '<div><strong>Status:</strong> ' + dbData.status + '</div>' +
                   '<div><strong>Type:</strong> ' + dbType + '</div>' +
                   '<div><strong>Driver:</strong> ' + driver + '</div>' +
                   '<div><strong>Mode:</strong> ' + modeText + '</div>' +
                   '</div>';
        }

        // Load content immediately
        Promise.all([
            fetch('/qr/status').then(response => response.json()),
            loadDatabaseStatus()
        ])
            .then(([qrData, dbData]) => {
                const content = document.getElementById('content');
                let qrContent = '';
                
                if (qrData.connected) {
                    qrContent = '<div class="status connected">✅ Successfully connected to WhatsApp!</div>';
                } else if (qrData.qr_available) {
                    qrContent = 
                        '<div class="status waiting">⏳ Waiting for QR code scan...</div>' +
                        '<div class="qr-container">' +
                        '<img src="/qr/image" alt="QR Code" class="qr-code" />' +
                        '</div>';
                } else {
                    qrContent = '<div class="status waiting">⏳ Generating QR code...</div>';
                }
                
                // Combine QR content with database status
                content.innerHTML = qrContent + renderDatabaseStatus(dbData);
            })
            .catch(err => {
                document.getElementById('content').innerHTML = 
                    '<div class="status waiting">⏳ Waiting for QR code...</div>';
            });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
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

// StartQRWebServer starts the QR web server
func (q *QRWebServer) StartQRWebServer(port int) {
	http.HandleFunc("/", q.ServeQRPage)
	http.HandleFunc("/qr/image", q.ServeQRImage)
	http.HandleFunc("/qr/status", q.ServeQRStatus)
	
	fmt.Printf("QR Web Server starting on http://localhost:%d\n", port)
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
