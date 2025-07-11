# mailhole

## REST API
| Method | Path                                         | Description                        |
|--------|----------------------------------------------|------------------------------------|
| GET    | `/emails/:recipient/messages`                | List all messages for recipient    |
| GET    | `/emails/:recipient/messages/:which`         | Get a specific message (by index, "first", or "last") |
| GET    | `/emails/:recipient/stream`                  | WebSocket stream for new messages  |

## Database Setup

1. **Start postgres and mailhole**
   ```sh
   docker compose up --build
2. **Install Atlas**
   [Install Atlas](https://atlasgo.io/getting-started/install) or run:
   ```sh
   curl -sSf https://atlasgo.sh | sh
3. **Create the database**
   ```sh
   psql -U postgres -c 'CREATE DATABASE mailhole;'
4. **Apply migrations**
   ```sh
   atlas migrate apply --dir file://db/migrations --url postgres://mailhole:mailhole@localhost:5432/mailhole?sslmode=disable
5. **(Optional) Baseline an existing database**
   ```sh
   atlas migrate apply --dir file://db/migrations --url postgres://mailhole:mailhole@localhost:5432/mailhole?sslmode=disable --baseline 20250709231344

## Try it

1. **In a different terminal, start the websocket**
   [Install wscat](https://github.com/websockets/wscat) or run:
   ```sh
   wscat -c ws://localhost:8080/emails/kevin@example.com/stream
2. **In a different terminal, send an email**
   ```sh
    curl -s "smtp://localhost:2525"   --mail-from "verify@example.com"   --mail-rcpt "kevin@example.com"   --upload-file testdata/email.txt
3. **Retrieve the last email to that address**
   ```sh
    curl -s "http://localhost:8080/emails/kevin@example.com/messages/last" | jq
    {
      "id": 307,
      "sender": "verify@example.com",
      "recipient": "kevin@example.com",
      "subject": "PaperlessPost Verification email",
      "body": "Please click here:\nhttps://example.com\nhttps://example.com/wedding\nhttps://example.com/pricing",
      "links": [
        "https://example.com",
        "https://example.com/wedding",
        "https://example.com/pricing"
      ],
      "received_at": "2025-07-11T19:57:33.460239Z"
    }
