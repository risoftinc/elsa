package constants

// Env manager command constants
const (
	EnvCommandUsage = "env"
	EnvCommandShort = "Manage local environment variables"
	EnvCommandLong  = `Local environment manager with a web UI.

Stores all data in a private SQLite database on your machine.
The web server binds to localhost only by default.`

	EnvServeUsage = "serve"
	EnvServeShort = "Start the environment manager web UI"
	EnvServeLong  = `Start a local web server for managing environment groups,
variables, and templates. Data is stored in SQLite under your user config directory.`

	EnvResetUsage = "reset"
	EnvResetShort = "Delete local env manager database (requires confirmation)"
	EnvResetLong  = `Permanently delete the local SQLite database and all stored
environments, variables, and templates.

You must type an 8-character random code shown in the terminal to confirm.
Stop "elsa env serve" before resetting if the database is in use.`

	DefaultEnvHost = "127.0.0.1"
	DefaultEnvPort = "1999"

	EnvFlagHost      = "host"
	EnvFlagPort      = "port"
	EnvFlagDB        = "db"
	EnvFlagNoBrowser = "no-browser"

	EnvFlagHostUsage      = "Host to bind (use 127.0.0.1 to keep data local)"
	EnvFlagPortUsage      = "Port for the web server"
	EnvFlagDBUsage        = "Path to SQLite database (default: user config dir)"
	EnvFlagNoBrowserUsage = "Do not open the system browser automatically"

	MsgEnvStarting   = "🌐 Elsa Env Manager: %s"
	MsgEnvDatabase   = "📁 Database: %s"
	MsgEnvPressCtrlC = "Press Ctrl+C to stop"

	MsgEnvResetWarning  = "⚠️  WARNING: This will permanently delete ALL env manager data."
	MsgEnvResetDatabase = "Database: %s"
	MsgEnvResetTypeCode = "Type this exact code to confirm (8 letters/digits, case-sensitive):\n%s"
	MsgEnvResetPrompt   = "> "
	MsgEnvResetAborted  = "Reset aborted — confirmation code did not match."
	MsgEnvResetSuccess  = "✅ Database deleted: %s"
	MsgEnvResetInUseHint = "💡 The database is still open. Press Ctrl+C in the terminal running \"elsa env serve\", then run reset again."

	ErrEnvResetMismatch = "confirmation code did not match"
)
