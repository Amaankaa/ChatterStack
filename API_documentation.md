# ChatterStack HTTP & WebSocket API Documentation

Base URL: `https://{host}:{port}/v1`

All non-auth endpoints require an `Authorization: Bearer {access_token}` header unless otherwise noted. Access tokens are short-lived JWTs issued by the auth endpoints.

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
  - `201 Created`: Returns the created user (without password hash).
  - `400 Bad Request`: Missing/invalid fields.
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
  - `200 OK`: Returns token pair `{ access_token, refresh_token }`.
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
  - `200 OK`: Returns new token pair.
  - `401 Unauthorized`: Invalid/expired refresh token.

### Logout
- **POST** `/auth/logout`
- **Request Body:**
  ```json
  {
    "user_id": "string"
  }
  ```
- **Responses:**
  - `204 No Content`: Session invalidated.
  - `400 Bad Request`: Missing user ID.


## User Endpoints (Protected)

### Get User by ID
- **GET** `/users/{userID}`
- **Responses:**
  - `200 OK`: Returns user profile.
  - `404 Not Found`: User missing.

### Get User by Email
- **GET** `/users?email={email}`
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
  - `404 Not Found`: Member/room missing.

### Search Rooms
- **GET** `/rooms?q={query}&limit={limit}`
- **Description:** Lists rooms the caller belongs to, optionally filtered by a partial name match.
- **Query Parameters:**
  - `q` *(optional)*: Substring to match on room names.
  - `limit` *(optional)*: Maximum number of results (default 20, max 100).
- **Responses:**
  - `200 OK`: `{ "rooms": [...] }`.
  - `400 Bad Request`: Invalid limit or other validation errors.


## Message Endpoints (Protected)

All message endpoints use the authenticated user ID from the bearer token.

### Send Message
- **POST** `/rooms/{roomID}/messages`
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
  - `201 Created`: Returns stored message.
  - `400 Bad Request`: Validation errors.

### List Messages
- **GET** `/rooms/{roomID}/messages`
- **Query Parameters:**
  - `page` *(optional)*: Defaults to 1 when paginating recent history.
  - `limit` *(optional)*: Max items per page/window (defaults to 50).
  - `around_message_id` *(optional)*: When supplied, ignores pagination and returns a window centered on the target message (see below).
- **Responses:**
  - `200 OK`: `{ "messages": [...] }`.
  - `400 Bad Request`: Invalid parameters or message not found in room.

**Around Message Window**

When `around_message_id` is provided, the service locates the target message, fetches half of the `limit` before it, the remainder after it, and returns the combined set chronologically. This enables “jump to message” experiences without paging through the entire timeline.

### Mark Delivered
- **POST** `/rooms/{roomID}/messages/{messageID}/deliver`
- **Responses:**
  - `204 No Content`: Delivery acknowledged.
  - `400 Bad Request`: Invalid message ID/user state.

### Mark Read
- **POST** `/rooms/{roomID}/messages/{messageID}/read`
- **Responses:**
  - `204 No Content`: Read receipt stored.
  - `400 Bad Request`: Invalid message ID/user state.

### Search Messages
- **GET** `/messages/search?q={query}&room_id={roomID}&limit={limit}`
- **Description:** Searches message content visible to the caller. Optionally filter to a single room.
- **Query Parameters:**
  - `q` *(required)*: Case-insensitive substring for message content.
  - `room_id` *(optional)*: Restrict search to a room.
  - `limit` *(optional)*: Maximum results (default 50, max 100).
- **Responses:**
  - `200 OK`: `{ "messages": [...] }`.
  - `400 Bad Request`: Missing `q` or invalid parameters.


## WebSocket Gateway

- **URL:** `ws(s)://{host}:{port}/ws?room_id=room-1&room_id=room-2`
- **Headers:** `Authorization: Bearer {access_token}`
- **Upgrade Params:** One or more `room_id` query params specify channels to join. The server verifies membership and rejects unauthorized rooms.

### Events (JSON)
- Outbound (server -> client):
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
      "created_at": "RFC3339 timestamp"
    }
  }
  ```
- Inbound (client -> server):
  ```json
  {
    "event": "send_message",
    "data": {
      "room_id": "string",
      "content": "string",
      "attachments": [...]
    }
  }
  ```

Messages sent via HTTP or WebSocket propagate through Redis pub/sub so every connected client in the room receives the `receive_message` event.

---

This document reflects the current server implementation. Adjust any base URLs or authentication flows to match your deployment environment.
