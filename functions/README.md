# Server functions

Place function files under this directory using the format:

`functions/<namespace>/<path>.mcfunction`

Example: `functions/minecraft/example.mcfunction` can be invoked with
`/function minecraft:example`.

Blank lines and lines beginning with `#` are ignored. The function executor
supports nested `/function` calls, rejects recursive cycles, limits nesting to
32 levels, and limits one execution to 10,000 command lines. Function files
must be at most 1 MiB.

The `/schedule` command can defer a function in the same world. Scheduled
calls are in-memory and are cleared on server shutdown.
