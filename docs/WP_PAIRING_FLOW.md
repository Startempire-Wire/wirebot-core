# WP Plugin: Pairing Auto‑Approve (Pseudo‑Flow)

```php
// On channel connect callback
function wirebot_on_channel_pairing($channel, $pairing_code, $user_id) {
    // Approve pairing in Clawdbot
    $cmd = sprintf(
        'clawdbot pairing approve %s %s',
        escapeshellarg($channel),
        escapeshellarg($pairing_code)
    );
    exec($cmd, $output, $status);

    if ($status !== 0) {
        error_log('Wirebot pairing approve failed: ' . implode("\n", $output));
    }
}

// Alternative: write allowFrom file directly
function wirebot_allowlist_user($channel, $user_id_or_handle) {
    $path = getenv('CLAWDBOT_STATE_DIR') . "/credentials/{$channel}-allowFrom.json";
    $data = [
        'version' => 1,
        'allowFrom' => [$user_id_or_handle]
    ];
    file_put_contents($path, json_encode($data, JSON_PRETTY_PRINT));
}
```

---

## See Also

- [PAIRING_ALLOWLIST.md](./PAIRING_ALLOWLIST.md) — Pairing + allowlist details
- [PLUGIN.md](./PLUGIN.md) — WordPress plugin spec
- [PROVISIONING.md](./PROVISIONING.md) — User provisioning
- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
