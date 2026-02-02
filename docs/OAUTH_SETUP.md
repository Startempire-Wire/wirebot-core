# OAuth App Setup Guide

Each OAuth provider requires creating an app on their developer console.
Once created, paste the Client ID and Secret into `/run/wirebot/scoreboard.env`.

## GitHub OAuth App

1. Go to: https://github.com/organizations/Startempire-Wire/settings/applications/new
2. Settings:
   - **Application name**: `Wirebot Scoreboard`
   - **Homepage URL**: `https://wins.wirebot.chat`
   - **Authorization callback URL**: `https://wins.wirebot.chat/v1/oauth/callback`
3. After creation, copy:
   - `Client ID` → `OAUTH_GITHUB_CLIENT_ID`
   - Generate `Client Secret` → `OAUTH_GITHUB_CLIENT_SECRET`

**Scopes requested**: `repo, admin:repo_hook` (read repos, manage webhooks)

## Stripe Connect

1. Go to: https://dashboard.stripe.com/settings/connect
2. Enable Connect platform (if not already)
3. Settings:
   - **Redirect URI**: `https://wins.wirebot.chat/v1/oauth/callback`
4. Copy:
   - `Client ID` (starts with `ca_`) → `OAUTH_STRIPE_CLIENT_ID`
   - Use your existing API secret key → `OAUTH_STRIPE_CLIENT_SECRET`

**Scopes requested**: `read_write` (read account data, create charges)

## Google (YouTube)

1. Go to: https://console.cloud.google.com/apis/credentials
2. Create OAuth 2.0 Client ID:
   - **Application type**: Web application
   - **Name**: `Wirebot Scoreboard`
   - **Authorized redirect URIs**: `https://wins.wirebot.chat/v1/oauth/callback`
3. Enable the **YouTube Data API v3** in API Library
4. Copy:
   - `Client ID` → `OAUTH_GOOGLE_CLIENT_ID`
   - `Client secret` → `OAUTH_GOOGLE_CLIENT_SECRET`

**Scopes requested**: `youtube.readonly` (read channel stats, video list)

## Applying Secrets

```bash
# Edit the scoreboard env file
vi /run/wirebot/scoreboard.env

# Add/update:
OAUTH_GITHUB_CLIENT_ID=Iv1_xxxxxxxxxxxx
OAUTH_GITHUB_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
OAUTH_STRIPE_CLIENT_ID=ca_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
OAUTH_STRIPE_CLIENT_SECRET=your_stripe_secret_key_here
OAUTH_GOOGLE_CLIENT_ID=xxxxxxxxxxxx-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com
OAUTH_GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# Restart scoreboard
systemctl restart wirebot-scoreboard
```

## Multi-Account Support

Each OAuth connection creates a separate integration row. Users can:
- Connect multiple GitHub accounts (personal + org)
- Connect multiple Stripe accounts (one per business)
- Connect multiple YouTube channels
- Tag each account with a business (STA, WIR, PHI, SEW)

The "+" Add button appears once at least one account is connected.

## For Sovereign Users (Tenants)

Each tenant operator creates their own OAuth apps on their own accounts.
The callback URL pattern is always: `https://{their-domain}/v1/oauth/callback`
Env vars are set per-tenant in their deployment configuration.
