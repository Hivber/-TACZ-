# Operator commands

This source fork registers a small operator command set in the executable:

- `/xuid` prints the authenticated Xbox Live XUID of the caller. It is available to all players.
- `/op <player>` grants operator permissions to an online player.
- `/deop <player>` revokes operator permissions from an online player.
- `/oplist` lists saved operator UUIDs.

The server process console uses Dragonfly's native command registry and
executor. Enter `op <online-player>` (a leading slash is optional),
`deop <online-player>`, or `oplist` directly in the terminal. Console commands
run as a trusted server-console source and do not require `OwnerXUID`.

## First-time owner setup

1. Start the server with Xbox Live authentication enabled.
2. Join using the account that should own the server and run `/xuid`.
3. Copy the returned XUID into `config.toml` under `[Server]` as `OwnerXUID = "..."`.
4. Restart the server. That account is now the bootstrap owner and can run `/op` and `/deop`.

If `OwnerXUID` is empty, no in-game player can grant or revoke operator permissions; the local server console can still administer operators. This prevents a regular player from self-promoting. Do not disable Xbox Live authentication on a public server.

Granted operators are stored by UUID in `operators.json`, which is created on the first grant. Their permissions are restored when they rejoin. Commands currently target online players; grant/revoke by offline name is not implemented.

The permission update packet is sent immediately. Dragonfly refreshes the command list on its normal background update cycle, so newly available operator commands may take a few seconds to appear in the client's command suggestions.
