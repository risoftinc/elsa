# Elsa - Engineer's Little Smart Assistant

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![Version](https://img.shields.io/badge/Version-1.0.0-green.svg)](https://github.com/risoftinc/elsa)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Elsa** is a powerful developer productivity toolkit for Go that provides database migration management, local environment variable management, file watching, custom command definitions, code generation, and project automation capabilities.

```
      ___           ___       ___           ___     
     /\  \         /\__\     /\  \         /\  \    
    /::\  \       /:/  /    /::\  \       /::\  \   
   /:/\:\  \     /:/  /    /:/\ \  \     /:/\:\  \  
  /::\~\:\  \   /:/  /    _\:\~\ \  \   /::\~\:\  \ 
 /:/\:\ \:\__\ /:/__/    /\ \:\ \ \__\ /:/\:\ \:\__\
 \:\~\:\ \/__/ \:\  \    \:\ \:\ \/__/ \/__\:\/:/  /
  \:\ \:\__\    \:\  \    \:\ \:\__\        \::/  / 
   \:\ \/__/     \:\  \    \:\/:/  /        /:/  /  
    \:\__\        \:\__\    \::/  /        /:/  /   
     \/__/         \/__/     \/__/         \/__/    V 1.0.0
(migration, scaffolding, project runner and task automation)
```

## 🚀 Key Features

### 📊 Database Migration Management
- **DDL/DML Separation**: Organized schema and data changes for production safety
- **Multi Database Support**: MySQL, PostgreSQL, SQLite
- **Migration Status Tracking**: View applied, pending, and rollback status
- **Flexible Naming**: Sequential or timestamp-based migration IDs
- **Production Ready**: Safe deployment with independent rollback control

### 👀 File Watching & Auto-Restart
- **Smart File Monitoring**: Watch Go files and auto-restart on changes
- **Configurable Extensions**: Customize which file types to monitor
- **Directory Exclusion**: Exclude vendor, build, and other directories
- **Restart Delays**: Configurable delays to prevent rapid restarts

### 📝 Elsafile - Custom Commands
- **Custom Command Syntax**: Define custom commands for your project
- **Command Management**: List, run, and manage custom commands
- **Conflict Detection**: Identify conflicts with built-in commands
- **Project Automation**: Streamline your development workflow

### 🔧 Code Generation & Scaffolding
- **Project Templates**: Generate boilerplate code and project structures
- **Template Caching**: Smart caching system with git-based cache paths
- **Cross-Platform Cache**: Platform-specific cache locations (Windows/macOS/Linux)
- **Git-Based Caching**: Cache paths follow git URL structure for better organization
- **Module Management**: Automatic go.mod module name creation

### 🗄️ DB Designer (Local)
- **Drag-and-drop canvas**: Design tables visually with PK/FK badges, indexes, and data types
- **Projects**: Separate workspaces per schema (MySQL, PostgreSQL, SQLite)
- **Relations**: FK lines with editable constraint name and ON DELETE / ON UPDATE actions
- **Indexes**: Composite and unique indexes per table, included in SQL import/export
- **Bidirectional SQL**: Live SQL panel; paste or edit SQL to update the diagram
- **Diagram export**: Preview then download PNG/SVG (dark or print-friendly layout)
- **Embedded UI**: Same dark theme as Env Manager, single binary

### 🔐 Env Manager (Local)
- **Web UI**: Embedded in a single binary
- **Environment Groups**: Custom groups (local, staging, production)
- **Variable Comparison**: Side-by-side table across up to 5 environments
- **Go Templates**: Export with `{{.KEY}}` syntax — copy or download `.env` files
- **Private Storage**: SQLite in user config dir; localhost-only by default

### 🏗️ Make System
- **Dynamic Template Types**: Generate files from configurable templates
- **Folder Structure Support**: Create files in custom directory structures
- **Template Versioning**: Support multiple template versions
- **Custom Template Override**: Override templates per project
- **Smart Template Resolution**: Priority-based template loading

## 📦 Installation

### Prerequisites
- Go 1.21 or higher
- Git

### Install from Repository
```bash
go install go.risoftinc.com/elsa/cmd/elsa@latest
```

### Verify Installation
```bash
elsa --version
```

## 🏃‍♂️ Quick Start

### 1. Initialize New Command
```bash
# Create new Elsafile for custom commands
elsa init

# This will create an Elsafile with common Go project commands
```

### 2. Database Migration
```bash
# Connect to your database (interactive setup)
elsa migration connect

# Or use direct connection string
elsa migration connect -c "mysql://user:password@localhost:3306/database"

# Create new DDL migration (schema changes)
elsa migration create ddl create_users_table

# Create new DML migration (data changes)
elsa migration create dml seed_users_data

# Apply migrations
elsa migration up ddl
elsa migration up dml

# Check migration status
elsa migration status
```

### 3. File Watching
```bash
# Watch Go files and auto-restart your application
elsa watch "go run main.go"

# Watch with custom settings
elsa watch "go test ./..." --ext ".go,.mod" --exclude "vendor,testdata"
```

### 4. Create New Project
```bash
# Create new project from xarch template
elsa new xarch my-api --module "github.com/username/my-api"

# Create with specific version
elsa new xarch@v1.2.3 my-api --module "github.com/username/my-api"

# Create with custom output directory
elsa new xarch my-api --module "github.com/username/my-api" --output "./projects"
```

**About xarch:**
[xarch](https://github.com/risoftinc/xarch) is a Go project template that provides a clean architecture structure. For detailed information about xarch and other templates, see the [Elsa New Guide](NEW_GUIDELINE.md).

### 5. Generate Files with Make
```bash
# Generate repository
elsa make repository user_repository

# Generate service
elsa make service user_service

# Generate with folder structure
elsa make repository health/health_repository

# List available template types
elsa make list
```

### 5. DB Designer
```bash
elsa dbdesign serve
```

See [DB Designer Guide](DBDESIGN_GUIDELINE.md).

### 6. Env Manager
```bash
# Start local env manager (opens browser automatically)
elsa env serve

# Custom port, skip browser open
elsa env serve --port 8080 --no-browser
```

See [Env Manager Guide](ENV_GUIDELINE.md) for environments, variables, templates, and export.

### 7. Custom Commands
```bash
# List available commands from Elsafile
elsa list

# Run custom command
elsa run build
elsa run test
```

## 📚 Commands Reference

### Root Commands
| Command | Description |
|---------|-------------|
| `elsa --help` | Show help information |
| `elsa --version` | Show version information |

### Migration Commands
| Command | Description |
|---------|-------------|
| `elsa migration connect` | Connect to database interactively |
| `elsa migration create ddl <name>` | Create DDL migration |
| `elsa migration create dml <name>` | Create DML migration |
| `elsa migration up <type>` | Apply migrations (type: `ddl` or `dml`) |
| `elsa migration down <type>` | Rollback last migration (type: `ddl` or `dml`) |
| `elsa migration status` | Show migration status |
| `elsa migration refresh <type>` | Refresh all migrations (type: `ddl` or `dml`) |

**Migration Types:**
- `ddl`: Data Definition Language (schema changes, table creation, modifications)
- `dml`: Data Manipulation Language (data seeding, updates, transformations)

**Why DDL/DML Separation?**
- **Production Safety**: Deploy schema and data changes independently
- **Rollback Control**: Rollback schema and data changes separately
- **Environment Consistency**: Schema changes apply consistently, data changes can be environment-specific
- **Team Collaboration**: Different team members can work on schema vs. data changes

### Watch Commands
| Command | Description |
|---------|-------------|
| `elsa watch <command>` | Watch files and auto-restart command |
| `--ext <extensions>` | File extensions to watch (default: .go) |
| `--exclude <dirs>` | Directories to exclude |
| `--delay <duration>` | Restart delay (e.g., 500ms, 1s) |

### Elsafile Commands
| Command | Description |
|---------|-------------|
| `elsa init` | Create new Elsafile |
| `elsa list` | List all commands |
| `elsa list --conflicts` | Show conflicting commands |
| `elsa run <command>` | Execute custom command |

### Generate Commands
| Command | Description |
|---------|-------------|
| `elsa generate` | Generate dependency injection code |
| `elsa gen` | Short alias for generate |

#### Quick Example

Create a file with `//go:build elsabuild` tag:

```go
//go:build elsabuild
// +build elsabuild

package http

import (
    "go.risoftinc.com/elsa"
    "gorm.io/gorm"
)

type Dependencies struct {
    UserHandler UserHandler
}

func InitializeHandler(db *gorm.DB) *Dependencies {
    elsa.Generate(RepositorySet, ServiceSet, HandlerSet)
    return nil
}

var RepositorySet = elsa.Set(NewUserRepository)
var ServiceSet = elsa.Set(NewUserService)
var HandlerSet = elsa.Set(NewUserHandler)
```

Run: `elsa generate` to create `elsa_gen.go` with automatic dependency injection.

### New Project Commands
| Command | Description |
|---------|-------------|
| `elsa new <template>[@version] <name>` | Create new project from template |
| `--module, -m` | Go module name (auto-generated if not provided) |
| `--output, -o` | Output directory (default: current) |
| `--force, -f` | Overwrite existing directory |
| `--refresh` | Force refresh template cache |

### DB Designer Commands
| Command | Description |
|---------|-------------|
| `elsa dbdesign serve` | Start visual schema designer web UI |
| `--host`, `--port`, `--db`, `--no-browser` | Same pattern as env manager (default port `1998`) |

### Env Manager Commands
| Command | Description |
|---------|-------------|
| `elsa env serve` | Start local env manager web UI |
| `elsa env reset` | Delete local database (requires 8-char confirmation code) |
| `--host` | Host to bind (default: `127.0.0.1`) |
| `--port` | HTTP port (default: `1999`) |
| `--db` | Custom SQLite database path |
| `--no-browser` | Do not open browser on start |

### Make Commands
| Command | Description |
|---------|-------------|
| `elsa make <template-type> <name>` | Generate file from template |
| `elsa make <template-type> <name> --refresh` | Generate file from template with fresh templates |
| `elsa make list` | List available template types |
| `elsa make --help` | Show make command help |

#### Template Types
The available template types are defined in your project's `.elsa-config.yaml` file. Common examples:
- `repository` - Generate repository layer files
- `service` - Generate service layer files
- `handler` - Generate HTTP/GRPC handler files

#### Examples
```bash
# Generate repository
elsa make repository user_repository

# Generate service
elsa make service user_service

# Generate with folder structure
elsa make repository health/health_repository

# Force refresh templates from remote repository
elsa make repository user_repository --refresh

# List available template types
elsa make list
```

#### Flags
- `--refresh`: Force refresh templates from remote repository, bypassing local cache

#### Configuration
The make system uses `.elsa-config.yaml` in your project root:
```yaml
source:
  git_url: https://github.com/risoftinc/xarch
  git_commit: abc123def456

make:
  repository:
    template: repository/template.go.tmpl
    output: domain/repositories
  service:
    template: service/template.go.tmpl
    output: domain/services
```

## 🔧 Configuration

### Elsafile Format
Create an `Elsafile` in your project root:

```bash
# Elsa - Engineer's Little Smart Assistant
# This file defines custom commands for your project
# Commands can be run using: elsa command_name or elsa run command_name

# Build the project
build:
	go build -o bin/app .

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run the application
run:
	go run .

# Format code
fmt:
	go fmt ./...
	go vet ./...
```

### Database Configuration
Elsa supports multiple connection methods:

#### Connection Priority Order
1. `-c` flag (highest priority)
2. `MIGRATE_CONNECTION` environment variable
3. `.env` file with `MIGRATE_CONNECTION` key
4. Individual database environment variables

#### Connection Examples
```bash
# Interactive setup (creates/updates .env file)
elsa migration connect

# Direct connection string
elsa migration connect -c "mysql://user:password@localhost:3306/database"
elsa migration connect -c "postgres://user:password@localhost:5432/database"
elsa migration connect -c "sqlite://database.db"

# Environment variable
export MIGRATE_CONNECTION="mysql://user:password@localhost:3306/database"
elsa migration up ddl
```

#### .env File
```env
# Single connection string (recommended)
MIGRATE_CONNECTION=mysql://user:password@localhost:3306/database
```

**Note**: When using `elsa migration connect` interactively, even if you input individual database details, the system automatically converts them to `MIGRATE_CONNECTION` format and saves it to `.env` file.

## 📁 Project Structure

```
your-project/
├── Elsafile                    # Custom commands definition
├── database/
│   └── migration/
│       ├── ddl/               # DDL migrations
│       │   ├── 20240101120000_create_users.up.sql
│       │   └── 20240101120000_create_users.down.sql
│       └── dml/               # DML migrations
│           ├── 20240101120001_seed_users.up.sql
│           └── 20240101120001_seed_users.down.sql
├── .env                       # Environment variables (optional)
└── your-go-files...
```

## 🎯 Use Cases

### Development Workflow
1. **Project Setup**: Use `elsa init` to create project commands
2. **Database Management**: Create and manage migrations
3. **Development**: Use `elsa watch` for auto-restart during development
4. **Testing**: Run tests with custom commands
5. **Deployment**: Use custom build and deployment commands

### Team Collaboration
- **Standardized Commands**: Share Elsafile across team members
- **Database Consistency**: Use migrations for schema changes
- **Development Environment**: Consistent setup across different machines

## 🤝 Contributing

We welcome contributions! 
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Setup
```bash
# Clone repository
git clone https://github.com/risoftinc/elsa.git
cd elsa

# Install dependencies
go mod download

# Run tests
go test ./...

# Build project
go build -o elsa ./cmd/elsa
```

## 📚 Documentation

### Detailed Guides
- **[Elsa Init & Run Breakdown](INIT_RUN_GUIDELINE.md)** - Complete guide for init and run commands
  - Command initialization and execution
  - Elsafile structure and syntax
  - Variable substitution and advanced features
  - Conflict resolution and best practices
- **[Elsa Watch Guide](WATCH_GUIDELINE.md)** - Complete file watching and auto-restart guide
  - Development workflow acceleration
  - Configuration options and best practices
  - Troubleshooting and performance optimization
  - Integration with development tools
- **[Elsa New Guide](NEW_GUIDELINE.md)** - Complete guide for creating new projects from templates
  - Template management and caching
  - Project creation process
  - Available templates (xarch)
  - Advanced features and troubleshooting
- **[Migration Guide](MIGRATION_GUIDELINE.md)** - Complete database migration management guide
  - DDL/DML separation rationale
  - Production deployment strategies
  - Best practices and troubleshooting
  - CI/CD integration examples
- **[Make System Guideline](MAKE_GUIDELINE.md)** - Complete guide for `elsa make` system
  - Template development
  - YAML configuration
  - Advanced usage patterns
  - Troubleshooting guide
- **[Elsa Generate Guide](GEN_GUIDELINE.md)** - Complete guide for dependency injection
  - Build tag system
  - Dependency definition
  - Advanced examples
  - Troubleshooting and best practices
- **[DB Designer Guide](DBDESIGN_GUIDELINE.md)** - Visual schema designer: tables, indexes, FK constraints, SQL sync, diagram export
- **[Env Manager Guide](ENV_GUIDELINE.md)** - Local environment variable manager
  - Environment groups, variables, and templates
  - Comparison table, export, and Go template syntax
  - Security, data storage, and REST API reference

### Command Reference
- `elsa --help` - General help
- `elsa make --help` - Make system help
- `elsa migration --help` - Migration commands
- `elsa watch --help` - File watching options
- `elsa env --help` - Env manager commands

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Links

- **Get to Know**: [https://risoftinc.com](https://risoftinc.com)
- **GitHub**: [https://github.com/risoftinc/elsa](https://github.com/risoftinc/elsa)
- **Issues**: [https://github.com/risoftinc/elsa/issues](https://github.com/risoftinc/elsa/issues)

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI framework
- Uses [GORM](https://gorm.io/) for database operations
- File watching powered by [fsnotify](https://github.com/fsnotify/fsnotify)

---

**Elsa** - Making Go development more productive, one command at a time! 🚀
