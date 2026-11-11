//! Authoritative player state used by the Rust gameplay core.

use serde::{Deserialize, Serialize};

#[derive(Clone, Copy, Debug, Default, PartialEq, Serialize, Deserialize)]
pub struct Position {
    pub x: f64,
    pub y: f64,
    pub z: f64,
    pub yaw: f32,
    pub pitch: f32,
}

#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
pub struct PlayerState {
    pub uuid: String,
    pub name: String,
    pub dimension: i32,
    pub game_mode: u8,
    pub position: Position,
    pub connected: bool,
    pub ticks_alive: u64,
}

impl Position {
    pub fn is_finite(self) -> bool {
        self.x.is_finite() && self.y.is_finite() && self.z.is_finite() && self.yaw.is_finite() && self.pitch.is_finite()
    }
}

impl PlayerState {
    pub fn new(uuid: String, name: String, dimension: i32, game_mode: u8, position: Position) -> Self {
        Self {
            uuid,
            name,
            dimension,
            game_mode,
            position,
            connected: true,
            ticks_alive: 0,
        }
    }

    pub fn apply_move(&mut self, dimension: i32, position: Position) {
        self.dimension = dimension;
        self.position = position;
    }

    pub fn tick(&mut self) {
        if self.connected {
            self.ticks_alive = self.ticks_alive.saturating_add(1);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn player_lifecycle_state_is_stable() {
        let mut player = PlayerState::new(
            "test-uuid".into(),
            "Test".into(),
            0,
            0,
            Position::default(),
        );
        player.apply_move(1, Position { x: 1.0, y: 64.0, z: -2.0, yaw: 90.0, pitch: 0.0 });
        player.tick();
        assert_eq!(player.dimension, 1);
        assert_eq!(player.position.x, 1.0);
        assert_eq!(player.ticks_alive, 1);
    }
}
