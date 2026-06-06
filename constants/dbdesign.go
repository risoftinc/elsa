package constants

// DB Design command constants
const (
	DBDesignCommandUsage = "dbdesign"
	DBDesignCommandShort = "Visual database schema designer"
	DBDesignCommandLong  = `Drag-and-drop database designer with PK/FK relations.
Projects are stored locally in SQLite. Export to SQL for MySQL, PostgreSQL, or SQLite.`

	DBDesignServeUsage = "serve"
	DBDesignServeShort = "Start the database designer web UI"
	DBDesignServeLong  = `Start a local web server for designing database schemas.
Data is stored in SQLite under your user config directory.`

	DefaultDBDesignHost = "127.0.0.1"
	DefaultDBDesignPort = "1998"

	DBDesignFlagHost      = "host"
	DBDesignFlagPort      = "port"
	DBDesignFlagDB        = "db"
	DBDesignFlagNoBrowser = "no-browser"

	DBDesignFlagHostUsage      = "Host to bind (use 127.0.0.1 to keep data local)"
	DBDesignFlagPortUsage      = "Port for the web server"
	DBDesignFlagDBUsage        = "Path to SQLite database (default: user config dir)"
	DBDesignFlagNoBrowserUsage = "Do not open the system browser automatically"

	MsgDBDesignStarting   = "🗄️ Elsa DB Designer: %s"
	MsgDBDesignDatabase   = "📁 Database: %s"
	MsgDBDesignPressCtrlC = "Press Ctrl+C to stop"
)
