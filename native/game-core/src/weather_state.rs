//! Selectively ported from MC-RS at da93a9501601fcdc284762fde88c24b44604991e.
//! Upstream: https://github.com/pedrokarim/mc-rs
//! License: MIT (see third_party/licenses/MC-RS-MIT.txt).
//! Weather state machine adapted to deterministic host-provided RNG.
use crate::random::{RandomSource, SplitMix64};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum WeatherKind { Clear, Rain, Thunder }

#[derive(Debug, Clone)]
pub struct WeatherState {
    pub current: WeatherKind,
    pub duration_ticks: u32,
}

pub const CLEAR_MIN: u32 = 12000;
pub const CLEAR_MAX: u32 = 168000;
pub const RAIN_MIN: u32 = 12000;
pub const RAIN_MAX: u32 = 24000;
pub const THUNDER_MIN: u32 = 3600;
pub const THUNDER_MAX: u32 = 15600;

impl WeatherState {
    pub fn new(rng: &mut impl RandomSource) -> Self {
        Self { current: WeatherKind::Clear, duration_ticks: rng.range_u32(CLEAR_MAX-CLEAR_MIN+1)+CLEAR_MIN }
    }

    pub fn tick(&mut self, rng: &mut impl RandomSource) {
        if self.duration_ticks > 0 { self.duration_ticks -= 1; } else { self.transition(rng); }
    }

    fn transition(&mut self, rng: &mut impl RandomSource) {
        self.current = match self.current {
            WeatherKind::Clear => WeatherKind::Rain,
            WeatherKind::Rain => if rng.next_f32() < 0.3 { WeatherKind::Thunder } else { WeatherKind::Clear },
            WeatherKind::Thunder => WeatherKind::Clear,
        };
        self.duration_ticks = match self.current {
            WeatherKind::Clear => rng.range_u32(CLEAR_MAX-CLEAR_MIN+1)+CLEAR_MIN,
            WeatherKind::Rain => rng.range_u32(RAIN_MAX-RAIN_MIN+1)+RAIN_MIN,
            WeatherKind::Thunder => rng.range_u32(THUNDER_MAX-THUNDER_MIN+1)+THUNDER_MIN,
        };
    }
}

impl Default for WeatherState {
    fn default() -> Self { Self::new(&mut SplitMix64::new(0)) }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tick_reduces_duration() {
        let mut rng = SplitMix64::new(123);
        let mut weather = WeatherState::new(&mut rng);
        let before = weather.duration_ticks;
        weather.tick(&mut rng);
        assert!(weather.duration_ticks < before || weather.current != WeatherKind::Clear);
    }

    #[test]
    fn seeded_weather_is_reproducible() {
        let mut a_rng = SplitMix64::new(42);
        let mut b_rng = SplitMix64::new(42);
        let mut a = WeatherState::new(&mut a_rng);
        let mut b = WeatherState::new(&mut b_rng);
        for _ in 0..20_000 {
            a.tick(&mut a_rng);
            b.tick(&mut b_rng);
            assert_eq!(a.current, b.current);
            assert_eq!(a.duration_ticks, b.duration_ticks);
        }
    }
}
