use dragonfly_game_core::ipc;

fn main() {
    let path = std::env::args().nth(1).unwrap_or_else(|| "/tmp/dragonfly-game-core.sock".into());
    if let Err(error) = ipc::serve(&path) {
        eprintln!("dragonfly-game-core: {error}");
        std::process::exit(1);
    }
}
