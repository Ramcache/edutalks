# Указываем PowerShell как shell (удобно для Windows)
set shell := ["powershell", "-NoProfile", "-Command"]

# Константы
MIGRATIONS_DIR := "./migrations"
DB_URL := "postgres://postgres:625325@localhost:5432/edutalks_db?sslmode=disable"
GOOSE := "go run github.com/pressly/goose/v3/cmd/goose@latest"
SWAG := "go run github.com/swaggo/swag/cmd/swag@latest"

# ==== MIGRATIONS ====

# Создание миграции
migrate-create name:
    if (-not "{{name}}") { echo "Usage: just migrate-create name=create_users_table"; exit 1 }
    {{GOOSE}} -dir {{MIGRATIONS_DIR}} create {{name}} sql



# Применить все миграции
migrate-up:
    {{GOOSE}} -dir {{MIGRATIONS_DIR}} postgres {{DB_URL}} up

# Откатить последнюю миграцию
migrate-down:
    {{GOOSE}} -dir {{MIGRATIONS_DIR}} postgres {{DB_URL}} down

# Просмотр состояния миграций
migrate-status:
    {{GOOSE}} -dir {{MIGRATIONS_DIR}} postgres {{DB_URL}} status

# Применить одну миграцию вверх
migrate-up-one:
    {{GOOSE}} -dir {{MIGRATIONS_DIR}} postgres {{DB_URL}} up-by-one

# Откатить одну миграцию вниз
migrate-down-one:
    {{GOOSE}} -dir {{MIGRATIONS_DIR}} postgres {{DB_URL}} down-by-one

# ==== SWAGGER ====

swag-init:
    {{SWAG}} init --parseDependency --parseInternal -g app/main.go

# ==== DEPLOY ====

deploy m b:
    if (-not "{{m}}" -or -not "{{b}}") { echo "Usage: just deploy m='commit msg' b=branch"; exit 1 }
    echo "🔧 Generating Swagger docs..."
    {{SWAG}} init --parseDependency --parseInternal -g app/main.go
    echo "🚀 Running DB migrations..."
    {{GOOSE}} -dir {{MIGRATIONS_DIR}} postgres {{DB_URL}} up
    echo "📦 Git add..."
    git add .
    echo "✅ Git commit..."
    git commit -m "{{m}}"
    echo "📤 Git push..."
    git push origin {{b}}
    echo "✅ Deploy complete."


# ==== INFO ====

info:
    echo ""
    echo "🛠 AVAILABLE COMMANDS:"
    echo "----------------------"
    echo "just deploy m=\"msg\" b=branch      🔄 Run swag, migrations, git add+commit+push"
    echo "just swag-init                     📚 Generate Swagger docs"
    echo "just migrate-create name=NAME      🛠  Create a new SQL migration"
    echo "just migrate-up                    ⬆️  Apply all migrations"
    echo "just migrate-down                  ⬇️  Rollback last migration"
    echo "just migrate-status                📊 Show migration status"
    echo "just migrate-up-one                ⬆️  Apply one migration"
    echo "just migrate-down-one              ⬇️  Rollback one migration"
