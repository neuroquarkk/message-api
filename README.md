# message-api

an independent implementation of an old GetStream backend engineering assignment built from scratch. the assignment is a small REST API for creating and retrieving messages and adding reactions using Go, PostgreSQL, Redis and Docker

## request flow

### `POST /messages`

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Postgres
    participant Redis

    Client->>API: POST /messages
    API->>Postgres: INSERT INTO messages
    Postgres-->>API: id, created_at
    API->>Redis: TxPipeline: LPUSH + LTRIM(0,9)
    alt pipeline fails
        API->>Redis: DEL cache:messages
    end
    API-->>Client: 201
```

### `GET /messages`

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Redis
    participant Postgres

    Client->>API: GET /messages?page=1

    alt page == 1
        API->>Redis: LRANGE cache:messages 0 9
        alt exactly 10 items, all parse OK
            Redis-->>API: cached messages
        else miss / partial / corrupt
            API->>Postgres: SELECT messages (paginated)
            Postgres-->>API: messages
        end
    else page > 1
        API->>Postgres: SELECT messages (paginated)
        Postgres-->>API: messages
    end

    API->>Postgres: SELECT reactions for page
    Postgres-->>API: reactions
    API-->>Client: messages + reactions + total count
```

paginated query: `ORDER BY created_at DESC, id DESC LIMIT 10 OFFSET ...`

### `POST /messages/{messageId}/reactions`

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Postgres

    Client->>API: POST /messages/{id}/reactions
    API->>Postgres: INSERT reaction (upsert on conflict)
    Postgres-->>API: id, score, created_at
    API-->>Client: 201
```

repeat reactions accumulate their score using an upsert on `(message_id, user_id, type)`

## Interpretations

> *"If the cache does not contain the requested messages, the application fetches the messages from the database"*
- **interpretation:** a dual-write to postgres and redis can partially fail causing the cache to desync from the db
- **implementation:** strict cache validation. if redis returns a partial page (`len < 10`), it is treated as a miss. if the redis pipeline fails during `POST`, the cache key is invalidated (`DEL`) so the next read falls back to the database truth

> *"Score... you can think of it as claps on Medium.com"*

* **interpretation:** repeat reactions from the same user for the same message and reaction type should be additive
* **implementation:** accumulate the score using `ON CONFLICT ... DO UPDATE`, incrementing the existing reaction count when the same `(message_id, user_id, type)` combination is submitted again


## API

| Method | Path                              | Body                                     | Query                | Success |
| ------ | --------------------------------- | ---------------------------------------- | -------------------- | ------- |
| `POST` | `/messages`                       | `message_text`, `user_id`                | —                    | `201`   |
| `GET`  | `/messages`                       | —                                        | `page` (default `1`) | `200`   |
| `POST` | `/messages/{messageId}/reactions` | `user_id`, `type`, `score` (default `1`) | —                    | `201`   |

## run

```bash
docker compose up -d --build
```
