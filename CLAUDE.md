# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Serenity is an open-source API-driven content management system. Named as the objective of the it is to be simple and flexible, which should allow for less troubles in building projects.

**Key Features:**
- Consistent REST API
- Hierarchal role based permissions
- Built-in authentication management using JWT
- Flexible entities and fields, there are zero assumptions on how the data in the CMS is going to be structured or used
- Endpoints delivering data will paginate results up to a limit
- Comprehensive error handling

## Common Development Commands

### Build & Run
```bash
make build                                      # Build the binary
go run main.go                                  # Run directly
```

### Testing
```bash
make test                                       # Run all tests
```

Integration tests use `testcontainers-go` to spin up a real Postgres instance automatically. Docker must be running locally to execute them.

### Code Quality
```bash
make pretty                                     # Format code with gofmt
make lint                                       # Lint with golangci-lint
```

### Cleanup
```bash
make clean                                      # Clean build artifacts
```

## Version Control

### Git Commit Message Format
All commit messages must follow Conventional Commit structure:

**Format:**
```
<type>[!]: <description>

[optional body]
```

**Rules:**
1. **Type**: All commits must start with a type and can be one of the following depending on the change being commited:
   - **feat**: A new feature
   - **fix**: A bug fix
   - **docs**: Documentation only changes
   - **styles**: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
   - **refactor**: A code change that neither fixes a bug nor adds a feature
   - **perf**: A code change that improves performance
   - **test**: Adding missing tests or correcting existing tests
   - **build**: Changes that affect the build system or external dependencies (example scopes: gulp, broccoli, npm)
   - **ci**:Changes to our CI configuration files and scripts (example scopes: Travis, Circle, BrowserStack, SauceLabs)
   - **chore**: Other changes that don't modify src or test files
   - **revert**: Reverts a previous commit
2. **Breaking Changes**: Any breaking changes must have an exclamation mark after the type and before the colon.
3. **Description**: A description must immediately follow the colon and space after the type. The description is a short summary of the code changes.
4. **Body**: A longer commit body may be provided after the short description, providing additional contextual information about the code changes. The body must begin one blank line after the description.

**Examples:**
```
feat!: send an email to the customer when a product is shipped
```

```
fix: prevent racing of requests

Introduce a request id and a reference to latest request. Dismiss incoming responses other than from latest request.

Remove timeouts which were used to mitigate the racing issue but are obsolete now.
```

## Architecture

### Tech Stack

- **Language**: Go (module: `github.com/nicholemattera/serenity`)
- **HTTP Router**: `chi` (`github.com/go-chi/chi/v5`)
- **Database**: PostgreSQL via `pgx` (`github.com/jackc/pgx/v5`)
- **Migrations**: `goose` (`github.com/pressly/goose/v3`) — SQL files embedded in the binary, run automatically on startup
- **Auth**: `golang-jwt/jwt/v5` for JWT, `golang.org/x/crypto` for bcrypt
- **Config**: `viper` (`github.com/spf13/viper`) — environment variable based, required vars validated at startup
- **Logging**: `slog` (stdlib) with JSON handler — use `slog.Error/Info` with key-value attributes, not `log.Printf`
- **Testing**: `testcontainers-go` for integration tests against real Postgres

### Design Philosophy

Serenity follows these core design principles:

1. **Security by Default**: Ensure only users who have the correct permissions are able to access routes and methods
2. **Consistency Over Cleverness**: All services follow similar patterns even when the underlying APIs differ significantly
3. **Environment-First Configuration**: Following Twelve-Factor App methodology for configuration
4. **Defensive Programming**: Required environment variables are checked at initialization, not at usage time
5. **Flexibility**: This is meant to be used across a wide variety of use cases

### Project Structure
```
Serenity/
├── main.go                          # Entry point — connects DB, runs migrations, starts server
├── go.mod                           # Go module definition
├── Makefile                         # Build and development commands
├── configs/
│   └── .env.example                 # Environment variable template
├── internal/
│   ├── config/                      # Viper-based env config, validated at startup
│   ├── database/
│   │   ├── database.go              # pgx connection pool + Migrate()
│   │   └── migrations/              # goose SQL migration files (embedded in binary)
│   ├── models/                      # Go structs for all domain types + shared Audit struct
│   └── repository/                  # Database access layer — one file per model
├── api/                             # OpenAPI/Swagger specs (future)
└── .github/                         # GitHub Actions workflows (future)
```

### Data Structure

#### Content

- **Composite**: This is the starting point and defines a piece of data. Composites have `default_read` and `default_write` boolean flags that control unauthenticated access — for example a form submission composite may have `default_write = true` so users don't need to be logged in to submit. Role-based permissions take precedence for authenticated users.
- **Field**: This is additional data related to the composite. Fields can be required, have a specific position in the composite, can have a default value, and other metadata. The field should also have a type, such as:
  - **Association**: This type is for having a field that is associated to another entity. Which composite this associate is for should be stored in the metadata for this field.
  - **Checkbox**: This type is for simple boolean data.
  - **Color**: This type is for storing color codes.
  - **Date**: This type is for storing date in ... format.
  - **DateTime**: This type is for storing date times in ... format.
  - **Dropdown**: This type is for storing a single selected item from a list of option. That list of options should be stored in the metadata for this field.
  - **Email**: This type is similar to the `ShortText` type, but has specific validation of email addresses.
  - **File**: This type is similar to the `ShortText` type, but is used for storing an identifier to a file that was uploaded.
  - **LongText**: This type is for storing plain text.
  - **Number**: This type is for storing numbers
  - **Phone**: This type is similar to the `ShortText` type, but has specific validation of phone numbers.
  - **ShortText**: This type is for storing plain text.
  - **Time**: This type is for storing time in ... format.
  - **URL**: This type is similar to the `ShortText` type, but has specific validation of URLs.
- **Entity**: This is a piece of data associated to a specific composite. Entities have human-readable slugs. Entities support cross-composite parent-child relationships (e.g. a Comment entity belonging to a Post entity) using the Nested Set Model scoped by `tree_id` rather than `composite_id`. Each root entity anchors its own tree (`tree_id = id`); child entities inherit the root's `tree_id`. Root entities within a composite are ordered by `root_position`. The `lft`/`rgt` columns use the reserved-word-safe names in SQL.
- **Field Value**: This is the value for a field associated to an entity and a field that the entity's composite has.

#### Users

- **Role**: This is the starting point for user management. Roles should be ordered hierarchically and there should be specific session timeouts per role. 
- **Permission**: These are associate to specific composite's and role's. A role can have read or write flags per composite.
- **User**: This is the user record, which contains a user's first and last name, email address, salted password hash and which role the user is associated with.

#### Auditing

We should be keeping track of when and who: created, last updated and deleted any of the objects in our data structure.