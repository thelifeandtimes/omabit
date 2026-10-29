package protocol

import "time"

// Code retrieval verifies engine/container ownership and brackets the secret read
// with two identity reads. Keep this bounded independently of ordinary five-second
// inventory RPCs; the transport allowance includes response/audit publication.
const LoginCodeReadTimeout = 12 * time.Second
const LoginCodeReplyTimeout = 15 * time.Second
