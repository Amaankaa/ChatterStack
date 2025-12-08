# ChatterStack HTTP & WebSocket API Documentation

Base URLs:
- CloudFront REST: `https://d1176qoi9kdya5.cloudfront.net/v1`
- ALB REST (direct): `http://chatterstack-alb-730649082.us-east-1.elb.amazonaws.com/v1`
- CloudFront WebSocket: `wss://d1176qoi9kdya5.cloudfront.net/ws`

Authentication headers:
- Through CloudFront (REST), send `X-Auth-Token: <access_jwt>`.
- Direct to ALB (REST), you may use `Authorization: Bearer <access_jwt>` or `X-Auth-Token: <access_jwt>`.
- WebSocket auth via query string: `access_token=<access_jwt>`.

Access tokens are JWTs issued by the auth endpoints and should be treated as short-lived.

## Authentication Endpoints

### Register
- **POST** `/auth/register`
- **Description:** Create a new user account.
- **Request Body:**
  ```json
  {
    "username": "string",
    "email": "string",
    "password": "string"
  }
  ```
- **Responses:**
  - `201 Created`: Returns the created user.
  - `400 Bad Request`: Validation errors (missing fields, invalid email, weak password).
  - `409 Conflict`: Email already in use.

### Login
- **POST** `/auth/login`
- **Request Body:**
  ```json
  {
    "email": "string",
    "password": "string"
  }
  ```
- **Responses:**
  - `200 OK`: Returns `{ "access_token": "...", "refresh_token": "..." }`.
  - `401 Unauthorized`: Invalid credentials.

### Refresh Token
- **POST** `/auth/refresh`
- **Request Body:**
  ```json
  {
    "refresh_token": "string"
  }
  ```
- **Responses:**
  - `200 OK`: Returns a new token pair.
  - `401 Unauthorized`: Refresh token missing, expired, or revoked.

### Logout
- **POST** `/auth/logout`
- **Headers:** `X-Auth-Token` (CloudFront) or `Authorization: Bearer` (ALB)
- **Responses:**
  - `204 No Content`: Session invalidated and refresh token revoked.

## User Endpoints (Protected)

### Get User by ID
- **GET** `/users/{userID}`
- **Responses:**
  - `200 OK`: Returns user profile.
  - `404 Not Found`: User missing.

### Get User by Email
- **GET** `/users?email={email}`
- **Headers:** `X-Auth-Token` (CloudFront) or `Authorization: Bearer` (ALB)
- **Responses:**
  - `200 OK`: Returns user profile.
  - `400 Bad Request`: Missing `email` query param.
  - `404 Not Found`: User missing.

### Update User Status
- **PATCH** `/users/{userID}/status`
- **Request Body:**
  ```json
  {
    "status": "ONLINE" | "OFFLINE"
  }
  ```
- **Responses:**
  - `204 No Content`: Status updated.
  - `400 Bad Request`: Invalid payload.
  - `404 Not Found`: User missing.

## Room Endpoints (Protected)

### Create Room
- **POST** `/rooms`
- **Headers:** `X-Auth-Token` (CloudFront) or `Authorization: Bearer` (ALB)
- **Request Body:**
  ```json
  {
    "name": "string",
    "is_group": true,
    "member_ids": ["string"],
    "creator_id": "string"
  }
  ```
- **Responses:**
  - `201 Created`: Returns room details.
  - `400 Bad Request`: Validation errors.

### List Room Members
- **GET** `/rooms/{roomID}/members`
- **Responses:**
  - `200 OK`: Returns list of members.
  - `404 Not Found`: Room missing.

### Add Room Member
- **POST** `/rooms/{roomID}/members`
- **Request Body:**
  ```json
  {
    "user_id": "string",
    "role": "ADMIN" | "MEMBER"
  }
  ```
- **Responses:**
  - `201 Created`: Member added.
  - `400 Bad Request`: Invalid payload.

### Remove Room Member
- **DELETE** `/rooms/{roomID}/members/{userID}`
- **Responses:**
  - `204 No Content`: Member removed.
  - `404 Not Found`: Member or room missing.

### Delete Room
- **DELETE** `/rooms/{roomID}`
- **Description:** Permanently removes the specified room along with memberships and messages.
- **Responses:**
  - `204 No Content`: Room deleted.
  - `403 Forbidden`: Caller is not the room creator.
  - `404 Not Found`: Room missing or already deleted.

### Search Rooms
- **GET** `/rooms?q={query}&limit={limit}`
- **Description:** Lists rooms the caller belongs to, optionally filtered by a partial name match.
- **Query Parameters:**
  - `q` *(optional)*: Substring to match on room names.
  - `limit` *(optional)*: Max results (default 20, max 100).
- **Responses:**
  - `200 OK`: `{ "rooms": [...] }`.
  - `400 Bad Request`: Invalid parameters.

### Ensure Direct Message Room *(Preview)*
- **POST** `/rooms/direct`
- **Description:** Idempotently returns a one-to-one room between the caller and the requested participant.
- **Request Body:**
  ```json
  {
    "peer_user_id": "string"
  }
  ```
- **Responses:**
  - `200 OK`: Existing direct room returned.
  - `201 Created`: Newly created direct room returned.
  - `400 Bad Request`: Invalid payload or attempting to DM yourself.
  - `404 Not Found`: Peer user missing or inaccessible.

## Message Endpoints (Protected)

All message endpoints infer the caller from the bearer token.

### Send Message
- **POST** `/rooms/{roomID}/messages`
- **Headers:** `X-Auth-Token` (CloudFront) or `Authorization: Bearer` (ALB)
- **Request Body:**
  ```json
  {
    "content": "string",
    "attachments": [
      {
        "url": "string",
        "mime_type": "string",
        "size_bytes": 123
      }
    ]
  }
  ```
- **Responses:**
  - `201 Created`: Returns the stored message, including `created_at` and `updated_at`.
  - `400 Bad Request`: Validation errors or unauthorized room access.

### List Messages
- **GET** `/rooms/{roomID}/messages`
- **Headers:** `X-Auth-Token` (CloudFront) or `Authorization: Bearer` (ALB)
- **Query Parameters:**
  - `page` *(optional)*: Defaults to 1 when paginating.
  - `limit` *(optional)*: Defaults to 50.
  - `around_message_id` *(optional)*: Returns a window centered on the supplied message.
- **Responses:**
  - `200 OK`: `{ "messages": [...] }`.
  - `400 Bad Request`: Invalid parameters or message not found.

When `around_message_id` is provided, half of the requested window is returned before the anchor and half after, allowing jump-to-message experiences without paging from the top.

### Edit Message
- **PATCH** `/rooms/{roomID}/messages/{messageID}`
- **Request Body:**
  ```json
  {
    "content": "string"
  }
  ```
- **Description:** Only the original sender can edit message text. Attachments are immutable.
- **Responses:**
  - `200 OK`: Returns the updated message with a refreshed `updated_at`.
  - `400 Bad Request`: Validation errors.
  - `403 Forbidden`: Caller is not the author.
  - `404 Not Found`: Message missing in the specified room.

### Delete Message
- **DELETE** `/rooms/{roomID}/messages/{messageID}`
- **Description:** Removes a message authored by the caller and broadcasts a deletion event.
- **Responses:**
  - `204 No Content`: Message deleted.
  - `403 Forbidden`: Caller is not the author.
  - `404 Not Found`: Message missing in the specified room.

### Mark Delivered
- **POST** `/rooms/{roomID}/messages/{messageID}/deliver`
- **Responses:**
  - `204 No Content`: Delivery acknowledged.
  - `400 Bad Request`: Invalid identifiers or state.

### Mark Read
- **POST** `/rooms/{roomID}/messages/{messageID}/read`
- **Responses:**
  - `204 No Content`: Read receipt stored.
  - `400 Bad Request`: Invalid identifiers or state.

### Search Messages
- **GET** `/messages/search?q={query}&room_id={roomID}&limit={limit}`
- **Description:** Searches message content the caller is authorized to view.
- **Query Parameters:**
  - `q` *(required)*: Case-insensitive substring.
  - `room_id` *(optional)*: Restrict search to a room.
  - `limit` *(optional)*: Defaults to 50, max 100.
- **Responses:**
  - `200 OK`: `{ "messages": [...] }`.
  - `400 Bad Request`: Missing `q` or invalid parameters.

### Message Resource Shape

Unless stated otherwise, message responses return:

```json
{
  "id": "string",
  "room_id": "string",
  "sender_id": "string",
  "content": "string",
  "attachments": [
    {
      "id": "string",
      "message_id": "string",
      "url": "string",
      "mime_type": "string",
      "size_bytes": 123
    }
  ],
  "status": "SENT" | "DELIVERED" | "READ",
  "created_at": "RFC3339 timestamp",
  "updated_at": "RFC3339 timestamp"
}
```

## WebSocket Gateway

- **URL:** `wss://d1176qoi9kdya5.cloudfront.net/ws?room_id=room-1&room_id=room-2`
- **Auth:** Provide token via query string `access_token=<access_jwt>`.
- **Query Params:** Supply one or more `room_id` values. The server verifies membership before joining rooms.

## CORS
- Allowed Origin: `https://chatterstack.vercel.app` (default). Configure `CORS_ALLOWED_ORIGINS` to add more (e.g., `https://localhost:3000`).
- Preflight (OPTIONS) responses include:
  - `Access-Control-Allow-Origin: https://chatterstack.vercel.app`
  - `Access-Control-Allow-Methods: GET,POST,PUT,PATCH,DELETE,OPTIONS`
  - `Access-Control-Allow-Headers: Content-Type, Authorization, X-Auth-Token`
  - `Access-Control-Allow-Credentials: true`
  - `Access-Control-Max-Age: 600`
- CloudFront `/v1/*` behavior forwards origin CORS headers without overriding them.

All inbound events are JSON objects with an `event` field and a `data` payload. Unknown events are ignored.

### Inbound Events (client → server)

#### `send_message`
```json
{
  "event": "send_message",
  "data": {
    "room_id": "string",
    "content": "string",
    "attachments": [
      { "url": "string", "mime_type": "string", "size_bytes": 123 }
    ]
  }
}
```

#### `typing_start` and `typing_stop`
```json
{
  "event": "typing_start", // or "typing_stop"
  "data": {
    "room_id": "string" // optional when the connection joined exactly one room
  }
}
```

Typing events are rate-limited server side. The sender is never echoed back to avoid flicker in the UI.

### Outbound Events (server → client)

#### `receive_message`
```json
{
  "event": "receive_message",
  "data": {
    "id": "string",
    "room_id": "string",
    "sender_id": "string",
    "content": "string",
    "attachments": [...],
    "status": "SENT" | "DELIVERED" | "READ",
    "created_at": "RFC3339 timestamp",
    "updated_at": "RFC3339 timestamp"
  }
}
```

#### `message.deleted`
Broadcast when a sender deletes one of their messages.

```json
{
  "event": "message.deleted",
  "data": {
    "id": "string",
    "room_id": "string"
  }
}
```

#### `typing_start` and `typing_stop`
```json
{
  "event": "typing_start", // or "typing_stop"
  "data": {
    "username": "string",
    "room_id": "string"
  }
}
```

Typing events are broadcast to all members in the room except the originator. The `username` field reflects the display name supplied during the WebSocket upgrade (falls back to the user ID when a profile lookup fails).

Messages sent over REST or WebSocket propagate through Redis pub/sub so every connected client in a room receives the matching outbound events in real time. Ensure your frontend filters duplicate messages if it performs optimistic updates for the same rooms.

---

This document reflects the current server implementation and CloudFront behavior. Adjust base URLs and authentication flows to match your deployment environment.
