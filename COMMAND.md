# Fix api_client.py — OAMP Backend

## Status: HTTP endpoints done ✅

All HTTP fixes applied:
- base_url default fixed
- list_rooms response key fixed
- submit_game added

## Remaining: WebSocket

WebSocket needed for real-time 1v1 match. Backend uses `/ws/match/:room_id?role={player|spectator}&player_id=X`.

### WebSocket endpoints

| URL | Purpose |
|-----|---------|
| `WS /ws/match/:room_id?role=player&player_id=P1` | Player connects, sends score/game_over |
| `WS /ws/match/:room_id?role=spectator&player_id=S1` | Spectator connects, receives broadcasts |

### Player → Server messages (JSON)

```python
# Throttled score update (~5 Hz)
{"game_score": 3, "blocks_hit": 12}

# Immediate collision hit (non-throttled)
{"type": "SCORE_UPDATE", "game_score": 3, "blocks_hit": 12}

# Game over — sent once, persisted to DB
{"type": "GAME_OVER", "game_score": 95, "blocks_hit": 15, "play_duration": 42.5}
```

### Server → Spectator broadcasts

```python
{"type": "join", "player_id": "P1"}
{"type": "score_update", "player_id": "P1", "game_score": 85, "blocks_hit": 12}
{"type": "GAME_OVER", "player_id": "P1", "game_score": 95, "blocks_hit": 15}
{"type": "leave", "player_id": "P1"}
```

### Add to api_client.py

Add `WSMatchClient` class:

```python
class WSMatchClient:
    """WebSocket client for real-time 1v1 match."""

    def __init__(self, ws_url: str, on_message: Optional[Callable] = None):
        self.ws_url = ws_url
        self.on_message = on_message
        self._ws = None
        self._recv_thread = None
        self._running = False

    def connect(self) -> bool:
        """Connect to WS endpoint. Returns True on success."""
        try:
            self._ws = websocket.WebSocketApp(
                self.ws_url,
                on_message=self._on_ws_message,
                on_error=self._on_ws_error,
                on_close=self._on_ws_close,
                on_open=self._on_ws_open,
            )
            self._running = True
            self._recv_thread = threading.Thread(target=self._ws.run_forever, daemon=True)
            self._recv_thread.start()
            return True
        except Exception as e:
            logger.error("WS connect failed: %s", e)
            return False

    def send_score(self, game_score: int, blocks_hit: int) -> None:
        """Send throttled score update."""
        self._send({"game_score": game_score, "blocks_hit": blocks_hit})

    def send_game_over(self, game_score: int, blocks_hit: int, play_duration: float) -> None:
        """Send game over — one shot, persisted to DB."""
        self._send({
            "type": "GAME_OVER",
            "game_score": game_score,
            "blocks_hit": blocks_hit,
            "play_duration": play_duration,
        })

    def _send(self, payload: dict) -> None:
        if self._ws:
            try:
                self._ws.send(json.dumps(payload))
            except Exception as e:
                logger.error("WS send failed: %s", e)

    def _on_ws_message(self, ws, msg) -> None:
        if self.on_message:
            try:
                data = json.loads(msg)
                self.on_message(data)
            except Exception as e:
                logger.error("WS message parse failed: %s", e)

    def _on_ws_error(self, ws, err) -> None:
        logger.error("WS error: %s", err)

    def _on_ws_close(self, ws, code, reason) -> None:
        logger.info("WS closed: %s %s", code, reason)
        self._running = False

    def _on_ws_open(self, ws) -> None:
        logger.info("WS connected: %s", self.ws_url)

    def close(self) -> None:
        self._running = False
        if self._ws:
            self._ws.close()
```

### Usage

```python
# Build WS URL from base_url
ws_url = base_url.replace("http", "ws") + f"/ws/match/{room_id}?role=player&player_id={player_id}"
ws = WSMatchClient(ws_url, on_message=lambda msg: print("spectator msg:", msg))
ws.connect()

# During play
ws.send_score(game_score=3, blocks_hit=12)

# On game over
ws.send_game_over(game_score=95, blocks_hit=15, play_duration=42.5)

# Done
ws.close()
```

### Required imports

Add to api_client.py:
```python
import websocket  # pip install websocket-client
import threading
import asyncio   # if async needed for game loop
from typing import Callable, Optional
```

## Submit game after WS game over

After sending `GAME_OVER` via WS, also call HTTP `submit_game` for DB persistence:

```python
api.submit_game(
    participant_id=participant_id,
    game_score=score,
    blocks_hit=blocks_hit,
    hand_tracking_status="active",
    play_duration=duration,
    timestamp=datetime.now(timezone.utc).isoformat(),
)
```

## Verification

Test with:
```bash
# List rooms
curl https://api.projectidek.dev/api/v1/rooms

# Create room
curl -X POST https://api.projectidek.dev/api/v1/rooms \
  -H "Content-Type: application/json" \
  -d '{"player_name":"test"}'
```