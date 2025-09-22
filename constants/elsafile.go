package constants

// Elsafile file constants
const (
	// DefaultElsafileName is the default filename for Elsafile
	DefaultElsafileName = "Elsafile"

	// DefaultFilePermissions is the default file permissions for creating Elsafile
	DefaultFilePermissions = 0644
)

// Built-in command constants
const (
	// BuiltinCommands contains all built-in Elsa commands that can conflict with Elsafile commands
	BuiltinCommands = "init,run,list,exec,migrate,watch,generate,gen,new,make,help,version"

	// CommandSeparator is used to join multiple commands in a single command definition
	CommandSeparator = " && "

	// CommandSuffix is the suffix that identifies a command definition line
	CommandSuffix = ":"

	// CommentPrefix is the prefix for comment lines
	CommentPrefix = "#"
)

// Duration formatting constants
const (
	// MillisecondThreshold is the threshold for displaying duration in milliseconds
	MillisecondThreshold = 1000

	// SecondThreshold is the threshold for displaying duration in seconds
	SecondThreshold = 60
)

// Error message constants
const (
	// ErrElsafileNotFound is the error message when Elsafile is not found
	ErrElsafileNotFound = "Elsafile not found at %s. Run 'elsa init' to create one"

	// ErrFailedToOpenFile is the error message when file opening fails
	ErrFailedToOpenFile = "failed to open Elsafile: %v"

	// ErrFailedToReadFile is the error message when file reading fails
	ErrFailedToReadFile = "error reading Elsafile: %v"

	// ErrCommandNotFound is the error message when a command is not found
	ErrCommandNotFound = "command '%s' not found in Elsafile"

	// ErrElsafileAlreadyExists is the error message when Elsafile already exists
	ErrElsafileAlreadyExists = "Elsafile already exists in current directory"

	// ErrFailedToWriteFile is the error message when file writing fails
	ErrFailedToWriteFile = "failed to write Elsafile: %v"

	// ErrCommandNameEmpty is the error message when command name is empty
	ErrCommandNameEmpty = "command name cannot be empty"

	// ErrCommandNameWhitespace is the error message when command name contains whitespace
	ErrCommandNameWhitespace = "command name cannot contain whitespace"

	// ErrCommandNameInvalidChars is the error message when command name contains invalid characters
	ErrCommandNameInvalidChars = "command name contains invalid characters"

	// ErrElsafileNotFoundInDirectories is the error message when Elsafile is not found in any directory
	ErrElsafileNotFoundInDirectories = "Elsafile not found in current or parent directories"
)

// Success message constants
const (
	// MsgElsafileCreatedSuccess is the success message when Elsafile is created
	MsgElsafileCreatedSuccess = "Created Elsafile successfully!"

	// MsgNoCommandsFound is the message when no commands are found
	MsgNoCommandsFound = "No commands found in Elsafile"

	// MsgNoConflictsFound is the message when no conflicts are found
	MsgNoConflictsFound = "No command conflicts found"
)

// Usage instruction constants
const (
	// UsageRunCommand is the usage instruction for running commands
	UsageRunCommand = "elsa run command_name    # Execute a command from Elsafile"

	// UsageListConflicts is the usage instruction for listing conflicts
	UsageListConflicts = "elsa list --conflicts    # Show conflicting commands"

	// UsageInit is the usage instruction for initialization
	UsageInit = "elsa init                # Create a new Elsafile"
)

// Default template constants
const (
	// DefaultTemplateHeader is the header comment for default Elsafile template
	DefaultTemplateHeader = `# Elsa - Engineer's Little Smart Assistant
# This file defines custom commands for your project
# Commands can be run using: elsa command_name or elsa run command_name`

	DefaultInstallToolsElsaCommand = `# Install elsa cli
install-tools:
	go install go.risoftinc.com/elsa/cmd/elsa@latest`

	// DefaultBuildCommand is the default build command template
	DefaultBuildCommand = `# Build the project
build:
	go build -o bin/app .`

	// DefaultTestCommand is the default test command template
	DefaultTestCommand = `# Run tests
test:
	go test ./...`

	// DefaultCleanCommand is the default clean command template
	DefaultCleanCommand = `# Clean build artifacts
clean:
	rm -rf bin/
	go clean`

	// DefaultDepsCommand is the default dependencies command template
	DefaultDepsCommand = `# Install dependencies
deps:
	go mod download
	go mod tidy`

	// DefaultRunCommand is the default run command template
	DefaultRunCommand = `# Run the application
start:
	go run .`

	// DefaultFmtCommand is the default format command template
	DefaultFmtCommand = `# Format code
fmt:
	go fmt ./...
	go vet ./...`
)

// Conflict resolution constants
const (
	// ConflictResolutionPrefix is the prefix used to resolve command conflicts
	ConflictResolutionPrefix = "run:"

	// ConflictResolutionPrefixLength is the length of the conflict resolution prefix
	ConflictResolutionPrefixLength = 4
)
