//! Persistent, versioned length-prefixed JSON IPC over a Unix domain socket.
use crate::{
    block_break::{BlockBreakProfile, MinerState, MiningTool},
    block_interaction::{BlockMiningMachine, BlockTarget, MiningError, MiningUpdate},
    inventory::Inventory,
    item_stack::ItemStack,
    player::{PlayerState, Position},
    CORE_PROTOCOL_VERSION,
};
use serde::{Deserialize, Serialize};
use std::{
    collections::HashMap,
    io::{Read, Write},
    os::unix::{fs::FileTypeExt, net::{UnixListener, UnixStream}},
    path::Path,
    sync::{Arc, Mutex},
    thread,
};

const MAX_FRAME: usize = 1024 *  1024;
const MAX_BATCH_EVENTS: usize = 4096;

#[derive(Debug, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
enum Request {
    Ping { protocol_version: u16 },
    Batch { protocol_version: u16, events: Vec<SemanticEvent> },
}

#[derive(Clone, Debug, Deserialize)]
#[serde(tag = "type", rename_all = "snake_case")]
enum SemanticEvent {
    PlayerJoin { uuid: String, name: String, dimension: i32, game_mode: u8, position: Position },
    PlayerMove { uuid: String, dimension: i32, position: Position },
    PlayerLeave { uuid: String },
    Tick,
    MiningStart {
        uuid: String, target: BlockTarget, profile: BlockBreakProfile,
        eye_position: [f32; 3], max_reach: f32, tick: u64,
    },
    MiningAdvance {
        uuid: String, target: BlockTarget, profile: BlockBreakProfile,
        tool: MiningTool, miner: MinerState, eye_position: [f32; 3],
        max_reach: f32, tick: u64,
    },
    MiningAbort { uuid: String },
    InventoryInit { inventory_id: String, slot_count: usize },
    InventorySync { inventory_id: String, slots: Vec<Option<ItemStack>> },
    InventoryInsert { inventory_id: String, stack: ItemStack },
    InventoryRemove { inventory_id: String, slot: usize, count: u16 },
    InventoryMove { inventory_id: String, source: usize, destination: usize, count: u16 },
}

#[derive(Debug, Serialize)]
#[serde(tag = "status", rename_all = "snake_case")]
enum MiningOutcome {
    Started { progress: f32, crack_stage: u8 },
    Progress { progress: f32, crack_stage: u8 },
    Completed,
    Aborted,
    Error { error: MiningError },
}

#[derive(Debug, Serialize)]
struct MiningResult { uuid: String, #[serde(flatten)] outcome: MiningOutcome }

#[derive(Debug, Serialize)]
#[serde(tag = "status", rename_all = "snake_case")]
enum InventoryOutcome {
    Snapshot { slots: Vec<Option<ItemStack>> },
    Inserted { remainder: Option<ItemStack>, slots: Vec<Option<ItemStack>> },
    Removed { removed: Option<ItemStack>, slots: Vec<Option<ItemStack>> },
    Updated { slots: Vec<Option<ItemStack>> },
    Error { error: String },
}

#[derive(Debug, Serialize)]
struct InventoryResult { inventory_id: String, #[serde(flatten)] outcome: InventoryOutcome }

#[derive(Debug, Serialize)]
struct Response {
    protocol_version: u16,
    ok: bool,
    message: &'static str,
    online_players: usize,
    processed_events: usize,
    mining_results: Vec<MiningResult>,
    inventory_results: Vec<InventoryResult>,
}

#[derive(Default)]
struct CoreState {
    players: HashMap<String, PlayerState>,
    mining: HashMap<String, BlockMiningMachine>,
    inventories: HashMap<String, Inventory>,
}

impl SemanticEvent {
    fn is_valid(&self) -> bool {
        match self {
            Self::PlayerJoin { uuid, name, position, .. } =>
                !uuid.is_empty() && !name.is_empty() && position.is_finite(),
            Self::PlayerMove { position, .. } => position.is_finite(),
            Self::PlayerLeave { uuid } | Self::MiningAbort { uuid } => !uuid.is_empty(),
            Self::InventoryInit { inventory_id, slot_count } => !inventory_id.is_empty() && *slot_count > 0 && *slot_count <= 256,
            Self::InventorySync { inventory_id, slots } => !inventory_id.is_empty() && !slots.is_empty() && slots.len() <= 256
                && slots.iter().flatten().all(ItemStack::is_valid),
            Self::InventoryInsert { inventory_id, stack } => !inventory_id.is_empty() && stack.is_valid(),
            Self::InventoryRemove { inventory_id, slot, count } => !inventory_id.is_empty() && *slot < 256 && *count > 0,
            Self::InventoryMove { inventory_id, source, destination, count } =>
                !inventory_id.is_empty() && *source < 256 && *destination < 256 && *count > 0,
            Self::Tick => true,
            Self::MiningStart { uuid, profile, eye_position, max_reach, .. } =>
                !uuid.is_empty() && profile.hardness.is_finite()
                    && eye_position.iter().all(|v| v.is_finite())
                    && max_reach.is_finite() && *max_reach > 0.0,
            Self::MiningAdvance { uuid, profile, tool, eye_position, max_reach, .. } =>
                !uuid.is_empty() && profile.hardness.is_finite()
                    && tool.efficiency_correct.is_finite()
                    && tool.efficiency_incorrect.is_finite()
                    && eye_position.iter().all(|v| v.is_finite())
                    && max_reach.is_finite() && *max_reach > 0.0,
        }
    }
}

impl CoreState {
    fn handle(&mut self, request: Request) -> Response {
        match request {
            Request::Ping { protocol_version } =>
                self.response(protocol_version == CORE_PROTOCOL_VERSION, "pong", 0, Vec::new(), Vec::new()),
            Request::Batch { protocol_version, events } => {
                if protocol_version != CORE_PROTOCOL_VERSION {
                    return self.response(false, "protocol_mismatch", 0, Vec::new(), Vec::new());
                }
                if events.len() > MAX_BATCH_EVENTS {
                    return self.response(false, "batch_too_large", 0, Vec::new(), Vec::new());
                }
                if !events.iter().all(SemanticEvent::is_valid) {
                    return self.response(false, "invalid_event", 0, Vec::new(), Vec::new());
                }
                let processed = events.len();
                let mut mining_results = Vec::new();
                let mut inventory_results = Vec::new();
                for event in events {
                    let (mining, inventory) = self.apply(event);
                    if let Some(result) = mining { mining_results.push(result); }
                    if let Some(result) = inventory { inventory_results.push(result); }
                }
                self.response(true, "batch_applied", processed, mining_results, inventory_results)
            }
        }
    }

    fn apply(&mut self, event: SemanticEvent) -> (Option<MiningResult>, Option<InventoryResult>) {
        match event {
            SemanticEvent::PlayerJoin { uuid, name, dimension, game_mode, position } => {
                self.players.insert(uuid.clone(), PlayerState::new(uuid, name, dimension, game_mode, position));
            }
            SemanticEvent::PlayerMove { uuid, dimension, position } => {
                if let Some(player) = self.players.get_mut(&uuid) {
                    player.apply_move(dimension, position);
                }
            }
            SemanticEvent::PlayerLeave { uuid } => {
                self.players.remove(&uuid);
                self.mining.remove(&uuid);
                self.inventories.retain(|inventory_id, _| !inventory_id.starts_with(&format!("player:{uuid}:")));
            }
            SemanticEvent::Tick => {
                for player in self.players.values_mut() {
                    player.tick();
                }
            }
            SemanticEvent::MiningStart { uuid, target, profile, eye_position, max_reach, tick } => {
                let machine = self.mining.entry(uuid.clone()).or_default();
                let outcome = match machine.start(target, profile, eye_position, max_reach, tick) {
                    Ok(MiningUpdate::Started { progress, crack_stage }) => MiningOutcome::Started { progress, crack_stage },
                    Ok(MiningUpdate::Progress { progress, crack_stage }) => MiningOutcome::Progress { progress, crack_stage },
                    Ok(MiningUpdate::Completed) => MiningOutcome::Completed,
                    Ok(MiningUpdate::Aborted) => MiningOutcome::Aborted,
                    Err(error) => MiningOutcome::Error { error },
                };
                return (Some(MiningResult { uuid, outcome }), None);
            }
            SemanticEvent::MiningAdvance { uuid, target, profile, tool, miner, eye_position, max_reach, tick } => {
                let machine = self.mining.entry(uuid.clone()).or_default();
                let outcome = match machine.advance(target, profile, tool, miner, eye_position, max_reach, tick) {
                    Ok(MiningUpdate::Started { progress, crack_stage }) => MiningOutcome::Started { progress, crack_stage },
                    Ok(MiningUpdate::Progress { progress, crack_stage }) => MiningOutcome::Progress { progress, crack_stage },
                    Ok(MiningUpdate::Completed) => MiningOutcome::Completed,
                    Ok(MiningUpdate::Aborted) => MiningOutcome::Aborted,
                    Err(error) => MiningOutcome::Error { error },
                };
                return (Some(MiningResult { uuid, outcome }), None);
            }
            SemanticEvent::MiningAbort { uuid } => {
                let outcome = self.mining.entry(uuid.clone()).or_default().abort();
                let outcome = match outcome {
                    MiningUpdate::Aborted => MiningOutcome::Aborted,
                    _ => unreachable!("abort always returns Aborted"),
                };
                return (Some(MiningResult { uuid, outcome }), None);
            }
            SemanticEvent::InventoryInit { inventory_id, slot_count } => {
                let inventory = Inventory::new(slot_count);
                let slots = inventory.slots().to_vec();
                self.inventories.insert(inventory_id.clone(), inventory);
                return (None, Some(InventoryResult { inventory_id, outcome: InventoryOutcome::Snapshot { slots } }));
            }
            SemanticEvent::InventorySync { inventory_id, slots } => {
                let mut inventory = Inventory::new(slots.len());
                for (slot, stack) in slots.into_iter().enumerate() {
                    if let Some(stack) = stack {
                        inventory.set(slot, Some(stack)).expect("validated inventory slot");
                    }
                }
                let snapshot = inventory.slots().to_vec();
                self.inventories.insert(inventory_id.clone(), inventory);
                return (None, Some(InventoryResult {
                    inventory_id,
                    outcome: InventoryOutcome::Snapshot { slots: snapshot },
                }));
            }
            SemanticEvent::InventoryInsert { inventory_id, stack } => {
                let outcome = match self.inventories.get_mut(&inventory_id) {
                    Some(inventory) => {
                        let remainder = inventory.insert(stack);
                        InventoryOutcome::Inserted { remainder, slots: inventory.slots().to_vec() }
                    }
                    None => InventoryOutcome::Error { error: "inventory_not_initialized".into() },
                };
                return (None, Some(InventoryResult { inventory_id, outcome }));
            }
            SemanticEvent::InventoryRemove { inventory_id, slot, count } => {
                let outcome = match self.inventories.get_mut(&inventory_id) {
                    Some(inventory) => match inventory.remove(slot, count) {
                        Ok(removed) => InventoryOutcome::Removed { removed, slots: inventory.slots().to_vec() },
                        Err(error) => InventoryOutcome::Error { error },
                    },
                    None => InventoryOutcome::Error { error: "inventory_not_initialized".into() },
                };
                return (None, Some(InventoryResult { inventory_id, outcome }));
            }
            SemanticEvent::InventoryMove { inventory_id, source, destination, count } => {
                let outcome = match self.inventories.get_mut(&inventory_id) {
                    Some(inventory) => match inventory.move_items(source, destination, count) {
                        Ok(()) => InventoryOutcome::Updated { slots: inventory.slots().to_vec() },
                        Err(error) => InventoryOutcome::Error { error },
                    },
                    None => InventoryOutcome::Error { error: "inventory_not_initialized".into() },
                };
                return (None, Some(InventoryResult { inventory_id, outcome }));
            }
        }
        (None, None)
    }

    fn response(&self, ok: bool, message: &'static str, processed_events: usize,
        mining_results: Vec<MiningResult>, inventory_results: Vec<InventoryResult>) -> Response {
        Response {
            protocol_version: CORE_PROTOCOL_VERSION, ok, message,
            online_players: self.players.len(), processed_events, mining_results, inventory_results,
        }
    }
}

fn read_frame(stream: &mut UnixStream) -> std::io::Result<Vec<u8>> {
    let mut header = [0u8; 4];
    stream.read_exact(&mut header)?;
    let len = u32::from_be_bytes(header) as usize;
    if len == 0 || len > MAX_FRAME {
        return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "invalid IPC frame length"));
    }
    let mut payload = vec![0; len];
    stream.read_exact(&mut payload)?;
    Ok(payload)
}

fn write_frame(stream: &mut UnixStream, payload: &[u8]) -> std::io::Result<()> {
    if payload.is_empty() || payload.len() > MAX_FRAME {
        return Err(std::io::Error::new(std::io::ErrorKind::InvalidData, "invalid IPC response length"));
    }
    stream.write_all(&(payload.len() as u32).to_be_bytes())?;
    stream.write_all(payload)?;
    stream.flush()
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
            Err(_) => state.lock().unwrap_or_else(std::sync::PoisonError::into_inner)
                .response(false, "invalid_request", 0, Vec::new(), Vec::new()),
        };
        write_frame(&mut stream, &serde_json::to_vec(&response).map_err(std::io::Error::other)?)?;
    }
}

pub fn serve(path: impl AsRef<Path>) -> std::io::Result<()> {
    let path = path.as_ref();
    if path.exists() {
        if !std::fs::symlink_metadata(path)?.file_type().is_socket() {
            return Err(std::io::Error::new(std::io::ErrorKind::AlreadyExists,
                "refusing to remove a non-socket IPC path"));
        }
        std::fs::remove_file(path)?;
    }
    let listener = UnixListener::bind(path)?;
    use std::os::unix::fs::PermissionsExt;
    std::fs::set_permissions(path, std::fs::Permissions::from_mode(0o600))?;
    let state = Arc::new(Mutex::new(CoreState::default()));
    for connection in listener.incoming() {
        match connection {
            Ok(stream) => {
                let state = Arc::clone(&state);
                thread::spawn(move || if let Err(e) = handle_connection(stream, state) {
                    eprintln!("dragonfly-game-core: IPC client: {e}");
                });
            }
            Err(error) => return Err(error),
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::block_break::tool_type;

    fn profile() -> BlockBreakProfile {
        BlockBreakProfile { hardness: 1.5, tool_type: tool_type::PICKAXE,
            harvest_level: 1, requires_correct_tool: true }
    }
    fn target() -> BlockTarget {
        BlockTarget { position: crate::block_interaction::BlockPos { x: 0, y: 0, z: 0 }, face: 1 }
    }
    fn eye() -> [f32; 3] { [0.5, 1.62, 3.0] }

    #[test]
    fn ipc_mining_start_returns_semantic_crack_state() {
        let mut state = CoreState::default();
        let response = state.handle(Request::Batch {
            protocol_version: 1,
            events: vec![SemanticEvent::MiningStart {
                uuid: "player-1".into(), target: target(), profile: profile(),
                eye_position: eye(), max_reach: 7.0, tick: 1,
            }],
        });
        assert!(response.ok);
        assert_eq!(response.mining_results.len(), 1);
        assert!(matches!(response.mining_results[0].outcome,
            MiningOutcome::Started { progress: 0.0, crack_stage: 0 }));
    }

    #[test]
    fn ipc_mining_advance_is_tick_idempotent_and_abort_clears_state() {
        let mut state = CoreState::default();
        let start = SemanticEvent::MiningStart {
            uuid: "player-1".into(), target: target(), profile: profile(),
            eye_position: eye(), max_reach: 7.0, tick: 10,
        };
        state.apply(start);
        let advance = || SemanticEvent::MiningAdvance {
            uuid: "player-1".into(), target: target(), profile: profile(),
            tool: MiningTool::default(), miner: MinerState { on_ground: true, ..Default::default() },
            eye_position: eye(), max_reach: 7.0, tick: 11,
        };
        let first = state.apply(advance()).0.unwrap();
        let duplicate = state.apply(advance()).0.unwrap();
        assert_eq!(serde_json::to_value(first).unwrap(), serde_json::to_value(duplicate).unwrap());
        let aborted = state.apply(SemanticEvent::MiningAbort { uuid: "player-1".into() }).0.unwrap();
        assert!(matches!(aborted.outcome, MiningOutcome::Aborted));
        let after_abort = state.apply(advance()).0.unwrap();
        assert!(matches!(after_abort.outcome, MiningOutcome::Error { error: MiningError::NoActiveSession }));
    }

    #[test]
    fn inventory_ipc_initializes_inserts_removes_and_cleans_up() {
        let mut state = CoreState::default();
        let init = state.handle(Request::Batch {
            protocol_version: 1,
            events: vec![SemanticEvent::InventoryInit { inventory_id: "player:player-1:carried".into(), slot_count: 2 }],
        });
        assert!(init.ok);
        assert_eq!(init.inventory_results.len(), 1);
        let stack = ItemStack::new("minecraft:cobblestone", 10, 0, vec![]).unwrap();
        let inserted = state.handle(Request::Batch {
            protocol_version: 1,
            events: vec![SemanticEvent::InventoryInsert { inventory_id: "player:player-1:carried".into(), stack }],
        });
        assert!(matches!(inserted.inventory_results[0].outcome,
            InventoryOutcome::Inserted { remainder: None, .. }));
        let removed = state.handle(Request::Batch {
            protocol_version: 1,
            events: vec![SemanticEvent::InventoryRemove {
                inventory_id: "player:player-1:carried".into(), slot: 0, count: 4,
            }],
        });
        assert!(matches!(removed.inventory_results[0].outcome,
            InventoryOutcome::Removed { removed: Some(ItemStack { count: 4, .. }), .. }));
        state.apply(SemanticEvent::PlayerLeave { uuid: "player-1".into() });
        assert!(!state.inventories.contains_key("player:player-1:carried"));
    }

    #[test]
    fn inventory_sync_atomically_replaces_the_complete_slot_snapshot() {
        let mut state = CoreState::default();
        state.apply(SemanticEvent::InventoryInit { inventory_id: "player:p:carried".into(), slot_count: 2 });
        let snapshot = vec![
            Some(ItemStack::new("minecraft:diamond", 3, 0, vec![]).unwrap()),
            None,
            ];
        let response = state.handle(Request::Batch {
            protocol_version: 1,
            events: vec![SemanticEvent::InventorySync { inventory_id: "player:p:carried".into(), slots: snapshot }],
        });
        assert!(response.ok);
        assert!(matches!(response.inventory_results[0].outcome, InventoryOutcome::Snapshot { .. }));
        let inventory = state.inventories.get("player:p:carried").unwrap();
        assert_eq!(inventory.len(), 2);
        assert_eq!(inventory.get(0).unwrap().identifier, "minecraft:diamond");
        assert!(inventory.get(1).is_none());
    }

    #[test]
    fn inventory_sync_rejects_invalid_or_oversized_snapshots_before_mutation() {
        let mut state = CoreState::default();
        state.apply(SemanticEvent::InventoryInit { inventory_id: "player:p:carried".into(), slot_count: 1 });
        let invalid = SemanticEvent::InventorySync {
            inventory_id: "player:p:carried".into(),
            slots: vec![Some(ItemStack {
                identifier: "minecraft:diamond_sword".into(), count: 2, metadata: 0, user_data: vec![],
            })],
        };
        assert!(!invalid.is_valid());
        let too_many = SemanticEvent::InventorySync {
            inventory_id: "player:p:carried".into(), slots: vec![None; 257],
        };
        assert!(!too_many.is_valid());
        assert_eq!(state.inventories.get("p").unwrap().len(), 1);
    }

    #[test]
    fn invalid_inventory_item_is_rejected_before_batch_mutation() {
        let mut state = CoreState::default();
        let response = state.handle(Request::Batch {
            protocol_version: 1,
            events: vec![
                SemanticEvent::InventoryInit { inventory_id: "player:p:carried".into(), slot_count: 9 },
                SemanticEvent::InventoryInsert {
                    inventory_id: "player:p:carried".into(),
                    stack: ItemStack { identifier: "minecraft:diamond_sword".into(),
                        count: 2, metadata: 0, user_data: vec![] },
                },
            ],
        });
        assert!(!response.ok);
        assert!(!state.inventories.contains_key("player:p:carried"));
    }

    #[test]
    fn invalid_batch_does_not_partially_apply_player_or_mining_events() {
        let mut state = CoreState::default();
        let events = vec![
            SemanticEvent::PlayerJoin { uuid: "u".into(), name: "p".into(),
                dimension: 0, game_mode: 0, position: Position::default() },
            SemanticEvent::MiningStart { uuid: "u".into(), target: target(), profile: profile(),
                eye_position: eye(), max_reach: f32::NAN, tick: 1 },
        ];
        assert!(!events.iter().all(SemanticEvent::is_valid));
        let response = state.handle(Request::Batch { protocol_version: 1, events });
        assert!(!response.ok);
        assert!(state.players.is_empty());
        assert!(state.mining.is_empty());
    }
}
