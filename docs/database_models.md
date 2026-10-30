# Database Models & Schemas

This document describes the high-level data models and storage strategies for the `engage-engine` Instagram-style engagement platform across **MongoDB**, **PostgreSQL**, **Redis**, and **Kafka**.

---

## 1. MongoDB (Primary Document Database)

MongoDB is the primary database for rich media, polymorphic content, and nested discussion threads.

### `posts` Collection
Stores shared user posts, multi-media arrays, and denormalized counter caches.

```json
{
  "_id": "ObjectId",
  "author_id": "string (UUID)",
  "caption": "string",
  "media_urls": ["string"],
  "media_type": "string (image | video | carousel)",
  "location": "string",
  "likes_count": 0,
  "comments_count": 0,
  "views_count": 0,
  "created_at": "ISODate",
  "updated_at": "ISODate"
}
```
**Indexes**:
- `{ "author_id": 1, "created_at": -1 }` (user profile post grids)
- `{ "created_at": -1 }` (global / discovery feeds)

---

### `comments` Collection
Stores comments and nested replies on posts.

```json
{
  "_id": "ObjectId",
  "post_id": "string (ObjectId)",
  "author_id": "string (UUID)",
  "parent_id": "string (ObjectId, nullable for top-level comments)",
  "content": "string",
  "likes_count": 0,
  "created_at": "ISODate",
  "updated_at": "ISODate"
}
```
**Indexes**:
- `{ "post_id": 1, "created_at": 1 }` (fetching post discussion threads)
- `{ "parent_id": 1 }` (fetching nested replies)

---

### `notifications` Collection
Stores user activity alerts (likes, comments, follows).

```json
{
  "_id": "ObjectId",
  "user_id": "string (UUID)",
  "actor_id": "string (UUID)",
  "type": "string (like | comment | follow)",
  "target_id": "string",
  "is_read": false,
  "created_at": "ISODate"
}
```
**Indexes**:
- `{ "user_id": 1, "is_read": 1, "created_at": -1 }` (user notification inbox)

---

## 2. PostgreSQL (Relational Database)

PostgreSQL provides strong consistency and relational integrity for user accounts and the social graph.

### `users` Table
```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(30) UNIQUE NOT NULL,
    email           VARCHAR(255) UNIQUE NOT NULL,
    display_name    VARCHAR(100),
    bio             TEXT,
    avatar_url      VARCHAR(512),
    website         VARCHAR(255),
    followers_count BIGINT DEFAULT 0,
    following_count BIGINT DEFAULT 0,
    posts_count     BIGINT DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_username ON users (username);
```

### `follows` Table
```sql
CREATE TABLE follows (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    follower_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followee_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status       VARCHAR(20) DEFAULT 'active',
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_follower_followee UNIQUE (follower_id, followee_id)
);

CREATE INDEX idx_follows_follower ON follows (follower_id);
CREATE INDEX idx_follows_followee ON follows (followee_id);
```

---

## 3. Redis (Cache, Fast Counters, & HyperLogLog)

Redis handles sub-millisecond lookups, live timeline caches, and probabilistic cardinality.

| Key Pattern | Redis Data Structure | Description |
| :--- | :--- | :--- |
| `likes:{target_type}:{target_id}` | `SET` | Set of `user_id`s who liked the target (`SISMEMBER` check, `SCARD` count) |
| `feed:{user_id}` | `ZSET` | User timeline containing `post_id`s scored by `created_at` timestamp |
| `analytics:hll:views:{post_id}` | `HyperLogLog` | Memory-efficient unique view counting via `PFADD` and `PFCOUNT` |
| `analytics:counter:views:{post_id}` | `String (Integer)` | Fast atomic counter incremented via `INCRBY` |
| `session:{token}` | `String` | Ephemeral user authentication session data with TTL |

---

## 4. Apache Kafka (Event Streaming Topics)

Kafka decouples high-volume writes and powers asynchronous worker consumers.

| Topic Name | Key | Payload Example | Purpose |
| :--- | :--- | :--- | :--- |
| `engage.views` | `post_id` | `{"target_id":"p1","viewer_id":"u2","ip":"...","timestamp":...}` | Ingests impressions; consumed by analytics worker |
| `engage.likes` | `post_id` | `{"user_id":"u1","target_id":"p1","target_type":"post","action":"like"}` | Triggers notifications and counter updates |
| `engage.comments` | `post_id` | `{"comment_id":"c1","post_id":"p1","author_id":"u2"}` | Triggers post comment counters and notification |
| `engage.follows` | `followee_id` | `{"follower_id":"u1","followee_id":"u2"}` | Triggers follow count updates and notification |
| `engage.fanout` | `author_id` | `{"post_id":"p1","author_id":"u1"}` | Triggers timeline feed push to active followers |

