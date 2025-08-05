# Supabase Authentication Setup

## Overview
The WhatsApp Bridge QR web interface now includes Supabase authentication to secure access to the QR code scanning page.

## Required Environment Variables

Set these environment variables before running the application:

```bash
export SUPABASE_URL="https://your-project.supabase.co"
export SUPABASE_JWT_SECRET="your-jwt-secret-from-supabase"
```

## Getting Your Supabase Credentials

1. **SUPABASE_URL**: 
   - Go to your Supabase project dashboard
   - Copy the "Project URL" from Settings → API

2. **SUPABASE_JWT_SECRET**: 
   - Go to Settings → API in your Supabase dashboard
   - Copy the "JWT Secret" (not the anon key)

## How It Works

### Development Mode
- If `SUPABASE_JWT_SECRET` is not set, authentication is **disabled**
- All routes work without login (for local development)

### Production Mode
- When environment variables are set, authentication is **required**
- Users must log in via Supabase to access QR codes
- JWT tokens are validated on each request

## Routes

- **Protected Routes** (require authentication):
  - `/` - Main QR code interface
  - `/qr/image` - QR code PNG image
  - `/qr/status` - QR status JSON

- **Public Routes** (no authentication):
  - `/login` - Login page
  - `/auth/callback` - Authentication callback

## Authentication Flow

1. User visits `/` (or any protected route)
2. If not authenticated, redirected to `/login`
3. User clicks "Login with Supabase"
4. Redirected to Supabase hosted auth
5. After successful login, redirected back to `/auth/callback`
6. Token stored in cookie, user redirected to main page

## Docker Deployment

Set environment variables when running the container:

```bash
docker run -e SUPABASE_URL="https://your-project.supabase.co" \
           -e SUPABASE_JWT_SECRET="your-jwt-secret" \
           -p 8080:8080 whatsapp-bridge
```

## Security Features

- JWT token validation using Supabase secret
- Secure cookie storage with `httpOnly` and `sameSite` flags
- Token expiration checking
- Automatic redirect to login for expired/invalid tokens
- Development mode bypass for local testing

## Testing

1. **Without auth** (development):
   ```bash
   go run main.go
   # Visit http://localhost:3000 - should work directly
   ```

2. **With auth** (production):
   ```bash
   export SUPABASE_URL="https://your-project.supabase.co"
   export SUPABASE_JWT_SECRET="your-jwt-secret"
   go run main.go
   # Visit http://localhost:3000 - should redirect to login
   ```
