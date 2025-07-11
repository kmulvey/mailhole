# mailhole

## Try it

```
docker compose up --build

# in a different terminal

wscat -c ws://localhost:8080/emails/kevin@example.com/stream

# in a different terminal

cd testdata

curl -s "smtp://localhost:2525"   --mail-from "verify@example.com"   --mail-rcpt "kevin@example.com"   --upload-file email.txt

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
```

## REST API
| Method | Path                                         | Description                        |
|--------|----------------------------------------------|------------------------------------|
| GET    | `/emails/:recipient/messages`                | List all messages for recipient    |
| GET    | `/emails/:recipient/messages/:which`         | Get a specific message (by index, "first", or "last") |
| GET    | `/emails/:recipient/stream`                  | WebSocket stream for new messages  |
