//! Persistent, versioned length-prefixed JSON IPC over a Unix domain socket.
use crate::{player::{PlayerState, Position}, CORE_PROTOCOL_VERSION};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, io::{Read, Write}, os::unix::{fs::FileTypeExt, net::{UnixListener, UnixStream}}, path::Path, sync::{Arc, Mutex}, thread};
const MAX_FRAME: usize = 1024 * 1024;
const MAX_BATCH_EVENTS: usize = 4096;

#[derive(Debug, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
enum Request { Ping { protocol_version: u16 }, Batch { protocol_version: u16, events: Vec<SemanticEvent> } }

#[derive(Clone, Debug, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
enum SemanticEvent {
    PlayerJoin { uuid: String, name: String, dimension: i32, game_mode: u8, position: Position },
    PlayerMove { uuid: String, dimension: i32, position: Position },
    PlayerLeave { uuid: String },
    Tick,
}

#[derive(Debug, Serialize)]
struct Response { protocol_version: u16, ok: bool, message: &'static str, online_players: usize, processed_events: usize }

#[derive(Default)]
struct CoreState { players: HashMap<String, PlayerState> }

impl CoreState {
    fn handle(&mut self, request: Request) -> Response {
        match request {
            Request::Ping { protocol_version } => self.response(protocol_version == CORE_PROTOCOL_VERSION, "pong", 0),
            Request::Batch { protocol_version, events } => {
                if protocol_version != CORE_PROTOCOL_VERSION { return self.response(false, "protocol_mismatch", 0); }
                if events.len() > MAX_BATCH_EVENTS { return self.response(false, "batch_too_large", 0); }
                let mut processed = 0;
                for event in events {
                    if !self.apply(event) { return self.response(false, "invalid_event", processed); }
                    processed += 1;
                }
                self.response(true, "batch_applied", processed)
            }
        }
    }
    fn apply(&mut self, event: SemanticEvent) -> bool {
        match event {
            SemanticEvent::PlayerJoin { uuid, name, dimension, game_mode, position } => {
                if uuid.is_empty() || name.is_empty() || !position.is_finite() { return false; }
                self.players.insert(uuid.clone(), PlayerState::new(uuid, name, dimension, game_mode, position));
            }
            SemanticEvent::PlayerMove { uuid, dimension, position } => {
                if !position.is_finite() { return false; }
                if let Some(player) = self.players.get_mut(&uuid) { player.apply_move(dimension, position); }
            }
            SemanticEvent::PlayerLeave { uuid } => { self.players.remove(&uuid); }
            SemanticEvent::Tick => { for player in self.players.values_mut() { player.tick(); } }
        }
        true
    }
    fn response(&self, ok: bool, message: &'static str, processed_events: usize) -> Response {
        Response { protocol_version: CORE_PROTOCOL_VERSION, ok, message, online_players: self.players.len(), processed_events }
    }
}
fn read_frame(stream: &mut UnixStream) -> std::io::Result<Vec<u8>> {
    let mut header = [0u8; 4]; stream.read_exact(&mut header)?;
    let len = u32::from_be_bytes(header) as usize;
    if len == 0 || len > MAX_FRAME { return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "invalid IPC frame length")); }
    let mut payload = vec![0; len]; stream.read_exact(&mut payload)?; Ok(payload)
}
fn write_frame(stream: &mut UnixStream, payload: &[u8]) -> std::io::Result<()> {
    if payload.is_empty() || payload.len() > MAX_FRAME { return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "invalid IPC response length")); }
    stream.write_all(&(payload.len() as u32).to_be_bytes())?;
    stream.write_all(payload)?; stream.flush()
}
fn handle_connection(mut stream: UnixStream, state: Arc<Mutex<CoreState>>) -> std::io::Result<()> {
    loop {
        let payload = match read_frame(&mut stream) {
            Ok(payload) => payload,
            Err(error) if error.kind() == std::io::ErrorKind::UnexpectedEof => return Ok(()),
            Err(error) => return Err(error),
        };
        let response = match serde_json::from_slice::<Request>(&payload) {
            Ok(request) => state.lock().unwrap_or_else(std::sync::PoisonError::into_inner).handle(request),
            Err(_) => state.lock().unwrap_or_else(std::sync::PoisonError::into_inner).response(false, "invalid_request", 0),
        };
        write_frame(&mut stream, &serde_json::to_vec(&response).map_err(std::io::Error::other)?)?;
    }
}
pub fn serve(path: impl AsRef<Path>) -> std::io::Result<()> {
    let path = path.as_ref();
    if path.exists() {
        if !std::fs::symlink_metadata(path)?.file_type().is_socket() {
            return Err(std::io::Error::new(std::io::ErrorKind::AlreadyExists, "refusing to remove a non-socket IPC path"));
        }
        std::fs::remove_file(path)?;
    }
    let listener = UnixListener::bind(path)?;
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600))?;
    let state = Arc::new(Mutex::new(CoreState::default()));
    for connection in listener.incoming() {
        match connection {
            Ok(stream) => { let state = Arc::clone(&state); thread::spawn(move || if let Err(e) = handle_connection(stream, state) { eprintln!("dragonfly-game-core: IPC client: {e}"); }); }
            Err(error) => return Err(error),
        }
    }
    Ok(())
}
#[cfg(test)]
mod tests {
    use super::*;
    fn pos() -> Position { Position { x: 1.0, y: 64.0, z: -2.0, yaw: 90.0, pitch: 0.0 } }
    #[test] fn rejects_protocol_version_mismatch() {
        let mut state = CoreState::default();
        assert!(!state.handle(Request::Ping { protocol_version: 99 }).ok);
    }
    #[test] fn tracks_batched_player_lifecycle_and_ticks() {
        let mut state = CoreState::default();
        let events = vec![SemanticEvent::PlayerJoin { uuid: "u".into(), name: "p".into(), dimension: 0, game_mode: 0, position: Position::default() }, SemanticEvent::Tick];
        let response = state.handle(Request::Batch { protocol_version: 1, events });
        assert!(response.ok); assert_eq!(response.processed_events, 2); assert_eq!(state.players["u"].ticks_alive, 1);
        assert!(state.handle(Request::Batch { protocol_version: 1, events: vec![SemanticEvent::PlayerMove { uuid: "u".into(), dimension: 1, position: pos() }] }).ok);
        assert_eq!(state.players["u"].dimension, 1);
        state.handle(Request::Batch { protocol_version: 1, events: vec![SemanticEvent::PlayerLeave { uuid: "u".into() }] });
        assert!(state.players.is_empty());
    }
    #[test] fn invalid_positions_are_rejected() {
        let mut state = CoreState::default();
        let bad = Position { x: f64::NAN, ..Position::default() };
        let response = state.handle(Request::Batch { protocol_version: 1, events: vec![SemanticEvent::PlayerJoin { uuid: "u".into(), name: "p".into(), dimension: 0, game_mode: 0, position: bad }] });
        assert!(!response.ok); assert!(state.players.is_empty());
    }
}