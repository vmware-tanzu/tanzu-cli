CREATE TABLE IF NOT EXISTS "tanzu_cli_operations"
(
    "cli_version"       TEXT NOT NULL,
    "os_name"           TEXT NOT NULL,
    "os_arch"           TEXT NOT NULL,
    "plugin_name"       TEXT,
    "plugin_version"    TEXT,
    "command"           TEXT NOT NULL,
    "cli_id"            TEXT NOT NULL,
    "command_start_ts"  TEXT NOT NULL,
    "command_end_ts"    TEXT NOT NULL,
    "target"            TEXT,
    "name_arg"          TEXT,
    "endpoint"          TEXT,
    "flags"             TEXT,
    "exit_status"       INTEGER,
    "is_internal"       TEXT,
    "error"             TEXT,
    PRIMARY KEY("cli_id","command","command_start_ts")
);