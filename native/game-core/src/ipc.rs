//! Versioned length-prefixed JSON IPC over a Unix domain socket.

use crate::{player::{PlayerState, Position}, CORE_PROTOCOL_VERSION};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, io::{Read, Write}, os::unix::net::{UnixListener, UnixStream}, path::Path};

const MAX_FRAME: usize = 1024 * 1024;

#[derive(Debug, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
enum Request {
    Ping { protocol_version: u16 },
    PlayerJoin { protocol_version: u16, uuid: String, name: String, dimension: i32, game_mode: u8, position: Position },
    PlayerMove { protocol_version: u16, uuid: String, dimension: i32, position: Position },
    PlayerLeave { protocol_version: u16, uuid: String },
    Tick { protocol_version: u16 },
}

#[derive(Debug, Serialize)]
struct Response {
    protocol_version: u16,
    ok: bool,
    message: &'static str,
    online_players: usize,
}

#[derive(Default)]
struct CoreState {
    players: HashMap<String, PlayerState>,
}

impl CoreState {
    fn handle(&mut self, request: Request) -> Response {
        let (version, message) = match request {
            Request::Ping { protocol_version } => (protocol_version, "pong"),
            Request::PlayerJoin { protocol_version, uuid, name, dimension, game_mode, position } => {
                self.players.insert(uuid.clone(), PlayerState::new(uuid, name, dimension, game_mode, position));
                (protocol_version, "player_joined")
            }
            Request::PlayerMove { protocol_version, uuid, dimension, position } => {
                if let Some(player) = self.players.get_mut(&uuid) { player.apply_move(dimension, position); }
                (protocol_version, "player_moved")
            }
            Request::PlayerLeave { protocol_version, uuid } => {
                self.players.remove(&uuid);
                (protocol_version, "player_left")
            }
            Request::Tick { protocol_version } => {
                for player in self.players.values_mut() { player.tick(); }
                (protocol_version, "tick")
            }
        };
        Response { protocol_version: CORE_PROTOCOL_VERSION, ok: version == CORE_PROTOCOL_VERSION, message, online_players: self.players.len() }
    }
}

fn read_frame(stream: &mut UnixStream) -> std::io::Result<Vec<u8>> {
    let mut header = [0u8; 4];
    stream.read_exact(&mut header)?;
    let len = u32::from_be_bytes(header) as usize;
    if len == 0 || len > MAX_FRAME { return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "invalid IPC frame length")); }
    let mut payload = vec![0; len];
    stream.read_exact(&mut payload)?;
    Ok(payload)
}

fn write_frame(stream: &mut UnixStream, payload: &[u8]) -> std::io::Result<()> {
    if payload.is_empty() || payload.len() > MAX_FRAME { return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "invalid IPC response length")); }
    stream.write_all(&(payload.len() as u32).to_be_bytes())?;
    stream.write_all(payload)?;
    stream.flush()
}

fn handle_connection(mut stream: UnixStream, state: &mut CoreState) -> std::io::Result<()> {
    let payload = read_frame(&mut stream)?;
    let response = match serde_json::from_slice::<Request>(&payload) {
        Ok(request) => state.handle(request),
        Err(_) => Response { protocol_version: CORE_PROTOCOL_VERSION, ok: false, message: "invalid_request", online_players: state.players.len() },
    };
    let bytes = serde_json::to_vec(&response).map_err(std::io::Error::other)?;
    write_frame(&mut stream, &bytes)
}

/// Serve the core until the process is stopped. The socket is local-only.
pub fn serve(path: impl AsRef<Path>) -> std::io::Result<()> {
    let path = path.as_ref();
    if path.exists() { std::fs::remove_file(path)?; }
    let listener = UnixListener::bind(path)?;
    let mut state = CoreState::default();
    for connection in listener.incoming() {
        match connection {
            Ok(stream) => { let _ = handle_connection(stream, &mut state); }
            Err(error) => return Err(error),
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_protocol_version_mismatch() {
        let mut state = CoreState::default();
        let response = state.handle(Request::Ping { protocol_version: 99 });
        assert!(!response.ok);
        assert_eq!(response.message, "pong");
    }

    #[test]
    fn tracks_player_join_move_leave_and_ticks() {
        let mut state = CoreState::default();
        let position = Position::default();
        let join = state.handle(Request::PlayerJoin { protocol_version: 1, uuid: "u".into(), name: "p".into(), dimension: 0, game_mode: 0, position });
        assert!(join.ok);
        state.handle(Request::Tick { protocol_version: 1 });
        assert_eq!(state.players["u"].ticks_alive, 1);
        state.handle(Request::PlayerLeave { protocol_version: 1, uuid: "u".into() });
        assert!(state.players.is_empty());
    }
}
