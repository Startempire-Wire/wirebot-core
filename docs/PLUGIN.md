# Wirebot WordPress Plugin Specification

> **The authoritative control, configuration, entitlement, and UI layer for Wirebot.**

---

## Plugin Overview

**Name:** `startempire-wirebot`

**Role:** The WordPress plugin that governs Wirebot access, trust, UX, and monetization.

The plugin does **not** do heavy AI work. The Gateway handles all intelligence.

The plugin **does**:
- Decide *who* can do *what*
- Decide *which trust mode* is allowed
- Issue credentials (JWT) to the Gateway
- Expose clean UX for non-technical founders
- Handle billing, consent, and compliance

---

## Core Responsibilities

### A. Identity & Trust Ceiling

Maps WordPress user → Wirebot founder ID.

Determines:
- Maximum trust mode (0 / 1 / 2 / 3)
- Allowed surfaces (web, SMS, extension, Discord)
- Enabled modules

```php
function get_founder_trust_ceiling($user_id) {
    $membership = get_user_membership_level($user_id);
    $advanced_flag = get_user_meta($user_id, 'wirebot_advanced', true);
    $sovereign_allowlist = get_option('wirebot_sovereign_users', []);
    
    if (in_array($user_id, $sovereign_allowlist)) {
        return 3; // Sovereign
    } elseif ($advanced_flag && $membership >= 'premium') {
        return 2; // Advanced
    } elseif ($membership >= 'basic') {
        return 1; // Standard
    }
    return 0; // Public
}
```

### B. JWT Issuer (Critical)

WordPress is the **issuer of authority**.

Uses `firebase/php-jwt` library.

```php
use Firebase\JWT\JWT;

function issue_wirebot_jwt($user_id, $workspace_id) {
    $secret = get_option('wirebot_jwt_secret');
    $trust_ceiling = get_founder_trust_ceiling($user_id);
    $scopes = get_founder_scopes($user_id);
    $surfaces = get_founder_surfaces($user_id);
    
    $payload = [
        'iss' => 'startempirewire.com',
        'sub' => $user_id,
        'workspace_id' => $workspace_id,
        'trust_mode_max' => $trust_ceiling,
        'scopes' => $scopes,
        'surfaces' => $surfaces,
        'iat' => time(),
        'exp' => time() + (15 * 60), // 15 minutes
    ];
    
    return JWT::encode($payload, $secret, 'HS256');
}
```

**JWT Claims:**

| Claim | Type | Description |
|-------|------|-------------|
| `iss` | string | Issuer (startempirewire.com) |
| `sub` | string | User ID |
| `workspace_id` | string | Active workspace |
| `trust_mode_max` | int | Maximum trust level (0-3) |
| `scopes` | array | Allowed operations |
| `surfaces` | array | Allowed access surfaces |
| `iat` | int | Issued at timestamp |
| `exp` | int | Expiration timestamp |

### C. Settings & Configuration UI

Clean admin UI for non-technical founders.

**Settings Panels:**

1. **General Settings**
   - Enable/disable Wirebot
   - Default workspace
   - Timezone

2. **Interaction Cadence**
   - Daily standup time
   - End-of-day reflection time
   - Weekly planning day

3. **SMS Settings**
   - Enable SMS
   - Phone verification
   - SMS cadence preferences

4. **Discord Settings**
   - Connect Discord account
   - Enable/disable Discord presence
   - Notification preferences

5. **Advanced Settings** (Mode 2+ only)
   - Extended memory
   - Beta features opt-in
   - Tool permissions

6. **Data & Privacy**
   - Data retention preferences
   - Export my data
   - Delete my data

**No JSON. No YAML. No "configure your agent."**

### D. Module Management

Each capability is a **module**, not a hard dependency.

```php
class Wirebot_Module {
    public $id;
    public $name;
    public $description;
    public $min_trust_mode;
    public $min_membership;
    public $settings_callback;
    public $capabilities;
}
```

**Core Modules:**

| Module | Description | Min Trust | Min Membership |
|--------|-------------|-----------|----------------|
| `core` | Basic chat and context | 1 | Basic |
| `accountability` | Standups, reflections, planning | 1 | Basic |
| `sms` | SMS check-ins and prompts | 1 | Basic |
| `discord` | Discord presence features | 1 | Premium |
| `advanced` | Extended memory, tools | 2 | Premium |
| `experimental` | Beta features | 2 | Premium |
| `sovereign` | Full access | 3 | Owner only |

Modules:
- Register capabilities
- Declare permission requirements
- Expose settings panels
- Are gated by membership + trust mode

### E. Data Ownership Boundary

**WordPress owns:**
- User consent
- Feature access
- Billing state
- High-level summaries (optional sync)
- Compliance data

**Gateway owns:**
- Execution
- Memory mechanics
- Reasoning
- Session state

This separation is intentional.

---

## Integration Points

### With Ring Leader Plugin

```php
// Consume identity from Ring Leader
$network_identity = ring_leader_get_user_identity($user_id);
$wirebot_founder_id = $network_identity['founder_id'];
```

### With Connect Plugin

```php
// Use Connect for extension authentication
add_filter('connect_extension_auth', function($auth_data, $user_id) {
    $auth_data['wirebot_jwt'] = issue_wirebot_jwt($user_id, get_default_workspace($user_id));
    return $auth_data;
}, 10, 2);
```

### With WebSockets Plugin (Optional)

```php
// Enable streaming responses via WebSockets
if (wirebot_websockets_enabled()) {
    add_action('wirebot_response_stream', 'websockets_broadcast_chunk');
}
```

---

## REST API Endpoints

```php
// Register REST routes
add_action('rest_api_init', function() {
    // Get JWT for current user
    register_rest_route('wirebot/v1', '/token', [
        'methods' => 'POST',
        'callback' => 'wirebot_get_token',
        'permission_callback' => 'is_user_logged_in',
    ]);
    
    // Get user settings
    register_rest_route('wirebot/v1', '/settings', [
        'methods' => 'GET',
        'callback' => 'wirebot_get_settings',
        'permission_callback' => 'is_user_logged_in',
    ]);
    
    // Update user settings
    register_rest_route('wirebot/v1', '/settings', [
        'methods' => 'POST',
        'callback' => 'wirebot_update_settings',
        'permission_callback' => 'is_user_logged_in',
    ]);
    
    // Get workspaces
    register_rest_route('wirebot/v1', '/workspaces', [
        'methods' => 'GET',
        'callback' => 'wirebot_get_workspaces',
        'permission_callback' => 'is_user_logged_in',
    ]);
});
```

---

## Database Tables (WordPress)

```sql
-- Wirebot user settings
CREATE TABLE {$wpdb->prefix}wirebot_settings (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    setting_key VARCHAR(100) NOT NULL,
    setting_value LONGTEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY user_setting (user_id, setting_key)
);

-- Wirebot workspaces (reference, synced from Gateway)
CREATE TABLE {$wpdb->prefix}wirebot_workspaces (
    id VARCHAR(36) PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    name VARCHAR(255),
    stage VARCHAR(20),
    is_default TINYINT(1) DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Module activations
CREATE TABLE {$wpdb->prefix}wirebot_modules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT UNSIGNED NOT NULL,
    module_id VARCHAR(50) NOT NULL,
    enabled TINYINT(1) DEFAULT 1,
    settings JSON,
    UNIQUE KEY user_module (user_id, module_id)
);
```

---

## File Structure

```
startempire-wirebot/
├── startempire-wirebot.php        # Main plugin file
├── includes/
│   ├── class-wirebot.php          # Core plugin class
│   ├── class-jwt-handler.php      # JWT generation/validation
│   ├── class-trust-manager.php    # Trust mode logic
│   ├── class-module-manager.php   # Module registration
│   ├── class-settings.php         # Settings management
│   └── class-rest-api.php         # REST endpoints
├── admin/
│   ├── class-admin.php            # Admin UI controller
│   ├── views/
│   │   ├── settings-general.php
│   │   ├── settings-sms.php
│   │   ├── settings-discord.php
│   │   └── settings-advanced.php
│   └── assets/
│       ├── css/
│       └── js/
├── modules/
│   ├── core/
│   ├── accountability/
│   ├── sms/
│   └── discord/
└── composer.json                  # Dependencies (firebase/php-jwt)
```

---

## Security Considerations

1. **JWT secret stored securely** — `get_option()` with encryption
2. **CSRF protection** — WordPress nonces on all forms
3. **Capability checks** — `current_user_can()` on all admin actions
4. **Input sanitization** — All user input sanitized
5. **Output escaping** — All output escaped
6. **Audit logging** — Critical actions logged
