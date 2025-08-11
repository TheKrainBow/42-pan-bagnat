#########################################################################################
#                                       CONFIG                                          #
#########################################################################################
NETWORK := pan-bagnat-net
DOCKER_COMPOSE := docker compose
POSTGRES_USER := ${POSTGRES_USER}
POSTGRES_PASSWORD := ${POSTGRES_PASSWORD}
POSTGRES_URL := postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost/panbagnat?sslmode=disable

#########################################################################################
#                                                                                       #
#                      DO NOT CHANGE ANYTHING AFTER THIS BLOCK                          #
#                                                                                       #
#########################################################################################
#                                         HELP                                          #
#########################################################################################
.PHONY: help
help:																					## Help | I am pretty sure you know what this one is doing!
	@printf "\033[1;34m📦 Makefile commands:\033[0m\n"
	@grep -E '^[a-zA-Z0-9_-]+:.*?##[A-Za-z0-9 _-]+\|.*$$' $(MAKEFILE_LIST) \
	| awk 'BEGIN {FS = ":.*?##|\\|"} \
	{ gsub(/^ +| +$$/, "", $$2); \
	  if (!seen[$$2]++) order[++i] = $$2; \
	  data[$$2] = data[$$2] sprintf("      \033[36m%-36s\033[0m %s\n", $$1, $$3) } \
	END { for (j = 1; j <= i; j++) { cat = order[j]; printf "   \033[32m%s\033[0m:\n%s", cat, data[cat] } }'

#########################################################################################
#                                        LOCAL                                          #
#########################################################################################
.PHONY: local-front local-back
local-front:																			## Local | Frontend locally (dev mode)
	cd frontend && BROWSER=none pnpm dev

local-back:																				## Local | Stop backend docker container, and run back locally (dev mode)
	$(DOCKER_COMPOSE) stop backend 
	cd backend/srcs && POSTGRES_URL=$(POSTGRES_URL) go run main.go

#########################################################################################
#                                      DATABASE                                         #
#########################################################################################
.PHONY: db-clear db-test db-clear-data db-init-schema

BE_WAS_RUNNING := $(shell docker inspect -f '{{.State.Running}}' pan-bagnat-backend-1 2>/dev/null)

stop-backend-if-needed:
	@if [ "$(BE_WAS_RUNNING)" = "true" ]; then \
	  echo "⛔ Stopping backend..."; \
	  docker stop pan-bagnat-backend-1; \
	fi

restart-backend-if-needed:
	@if [ "$(BE_WAS_RUNNING)" = "true" ]; then \
	  echo "▶️  Restarting backend..."; \
	  docker start pan-bagnat-backend-1; \
	fi

db-prune:																				
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	docker exec pan-bagnat-db-1 bash -lc "dropdb -U admin --if-exists schema_template"

db-init-schema: db-prune
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat < ./db/init/01_init.sql
	docker exec -i pan-bagnat-db-1 \
	  bash -lc "/docker-entrypoint-initdb.d/02_make_template.sh"

db-clear-data:
	docker exec -i pan-bagnat-db-1 \
	  psql -U admin -d panbagnat -c "\
	    TRUNCATE module_roles, user_roles, modules, roles, users RESTART IDENTITY CASCADE;\
	  "

db-prune-safe: stop-backend-if-needed db-prune restart-backend-if-needed				## Database | Prune database datas, schemas and templates
db-init-schema-safe: stop-backend-if-needed db-init-schema restart-backend-if-needed	## Database | Push schema to database
db-clear-data-safe: stop-backend-if-needed db-clear-data restart-backend-if-needed		## Database | Clear database datas

db-test: db-clear-data																	## Database | Set database datas with test datas
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat < ./db/test_data.sql

#########################################################################################
#                                       DOCKER                                          #
#########################################################################################
.PHONY: up up-dev down build build-back build-front prune fprune
up:																						## Docker Core | Up docker containers.   (PB only)
	@echo "⏳ Ensuring network '$(NETWORK)' exists…"
	@if ! docker network ls --format '{{.Name}}' | grep -q "^$(NETWORK)$$" ; then \
		echo "✅ Creating network '$(NETWORK)'"; \
		docker network create --driver bridge $(NETWORK); \
	else \
		echo "✅ Network '$(NETWORK)' already exists"; \
	fi
	@echo "🚀 Bringing up services…"
	$(DOCKER_COMPOSE) up -d

up-modules:																				## Docker Modules | Up docker containers.   (Modules only)
	@echo "⏳ Ensuring network '$(NETWORK)' exists…"
	@if ! docker network ls --format '{{.Name}}' | grep -q "^$(NETWORK)$$" ; then \
		echo "✅ Creating network '$(NETWORK)'"; \
		docker network create --driver bridge $(NETWORK); \
	else \
		echo "✅ Network '$(NETWORK)' already exists"; \
	fi
	@echo "🚀 Bringing up services…"
	@for dir in repos/*; do \
		if [ -d $$dir ] && [ -f $$dir/docker-compose.yml ]; then \
			echo "==> Stopping containers in $$dir"; \
			(cd $$dir && docker compose up); \
		fi \
	done

up-all: up up-modules																	## Docker Both | Up docker containers.   (PB & Modules)
down-all: down down-modules																## Docker Both | Stop docker containers. (PB & Modules)
stop-all: stop stop-modules																## Docker Both | Down docker containers. (PB & Modules)

stop:																					## Docker Core | Stop docker containers. (PB only)
	$(DOCKER_COMPOSE) stop


stop-modules:																			## Docker Modules | Stop docker containers. (Modules only)
	@for dir in repos/*; do \
		if [ -d $$dir ] && [ -f $$dir/docker-compose.yml ]; then \
			echo "==> Stopping containers in $$dir"; \
			(cd $$dir && docker compose stop); \
		fi \
	done

down:																					## Docker Core | Down docker containers. (PB only)
	$(DOCKER_COMPOSE) down

down-modules:																			## Docker Modules | Down docker containers. (Modules only)
	@for dir in repos/*; do \
		if [ -d $$dir ] && [ -f $$dir/docker-compose.yml ]; then \
			echo "==> Stopping containers in $$dir"; \
			(cd $$dir && docker compose down); \
		fi \
	done

prune:																					## Docker Core | Delete created images (PB & Modules)
	@echo "🛑 Bringing down containers…"
	$(DOCKER_COMPOSE) down
	@echo "🗑  Pruning images…"
	docker image prune -f
	@echo "🔍 Checking network '$(NETWORK)' usage…"
	@if docker network inspect $(NETWORK) > /dev/null 2>&1; then \
		if [ "$$(docker network inspect $(NETWORK) --format '{{len .Containers}}')" -eq "0" ]; then \
			echo "🗑  No containers attached—removing network '$(NETWORK)'"; \
			docker network rm $(NETWORK); \
		else \
			echo "ℹ️  Network '$(NETWORK)' still in use—skipping removal"; \
		fi \
	else \
		echo "⚠️  Network '$(NETWORK)' does not exist—nothing to do."; \
	fi

build: 																					## Docker Core | Build and up docker images. (PB only)
	$(DOCKER_COMPOSE) build
	@$(MAKE) -s up

build-back: 																			## Docker Core | Build and up backend docker images. (PB only)
	$(DOCKER_COMPOSE) build backend
	@$(MAKE) -s up

build-front: 																			## Docker Core | Build and up frontend docker images. (PB only)
	$(DOCKER_COMPOSE) build frontend
	@$(MAKE) -s up

REPO_DIRS := $(wildcard repos/*)

fprune: prune																			## Docker Core | Stop all containers, volumes, and networks. (PB & Modules)
	$(DOCKER_COMPOSE) down --volumes --remove-orphans || true
	docker network rm pan-bagnat_default 2>/dev/null || true
	docker system prune -af --volumes || true
	@echo $(REPO_DIRS)
	@for dir in $(REPO_DIRS); do \
	  if [ -d $$dir ]; then \
	    echo "==> fprune in $$dir"; \
	    ( \
	      cd "$$dir" && \
	      $(DOCKER_COMPOSE) down --volumes --remove-orphans || true; \
	      docker network rm pan-bagnat_default 2>/dev/null || true; \
	      docker system prune -af --volumes || true; \
	    ); \
	  fi; \
	done; \
	rm -rf $(REPO_DIRS)

#########################################################################################
#                                       TESTS                                           #
#########################################################################################
.PHONY: test-backend test-backend-verbose
test-backend:																			## Tests | Start tests for backend
	@echo "🧪 Running backend tests…"
	cd backend/srcs && POSTGRES_URL=$(POSTGRES_URL) go test -timeout 30s ./...

test-backend-verbose: 																	## Tests | Start tests for backend with verbose enabled
	@echo "🧪 Running backend tests…"
	cd backend/srcs && POSTGRES_URL=$(POSTGRES_URL) go test -v -timeout 30s ./...

#########################################################################################
#                                       SWAGGER                                         #
#########################################################################################
HOST_NAME ?= localhost:8080
swagger:
	cd backend/srcs && \
	swag init -g main.go --parseInternal && \
	sed -i 's/{{HOST_PLACEHOLDER}}/$(HOST_NAME)/' ./docs/docs.go && \
	sed -i 's/{{HOST_PLACEHOLDER}}/$(HOST_NAME)/' ./docs/swagger.json && \
	sed -i 's/{{HOST_PLACEHOLDER}}/$(HOST_NAME)/' ./docs/swagger.yaml

#########################################################################################
#                                      MIGRATIONS                                       #
#########################################################################################
# Config (override if needed)
ENV_FILE        ?= .env
MIGRATE_NETWORK ?= pan-bagnat-core
MIGRATIONS_DIR  ?= db/migrations
MIGRATE_IMAGE   ?= migrate/migrate:latest
POSTGRES_URL    ?= ${POSTGRES_URL}

# Internal: docker-run wrapper; expands $$POSTGRES_URL inside the container
define _MIGRATE_RUN
bash -lc 'set -a; source $(ENV_FILE); \
docker run --rm \
  --network $(MIGRATE_NETWORK) \
  -v "$(PWD)/$(MIGRATIONS_DIR)":/migrations:ro \
  -e POSTGRES_URL \
  $(MIGRATE_IMAGE) \
  -path=/migrations \
  -database "$$POSTGRES_URL" \
  $(1)'
endef

.PHONY: migrate-up migrate-down1 migrate-steps migrate-goto migrate-version migrate-force migrate-new

migrate-up:                                  ## Migrations | Apply all pending migrations (up)
	@$(call _MIGRATE_RUN,up)

migrate-down1:                               ## Migrations | Roll back the last migration (down 1)
	@$(call _MIGRATE_RUN,down 1)

migrate-steps:                               ## Migrations | Move N steps (use N=-2 to roll back two)
	@if [ -z "$(N)" ]; then echo "Usage: make migrate-steps N=-2"; exit 2; fi
	@$(call _MIGRATE_RUN,steps $(N))

migrate-goto:                                ## Migrations | Go to version V (e.g., V=2)
	@if [ -z "$(V)" ]; then echo "Usage: make migrate-goto V=2"; exit 2; fi
	@$(call _MIGRATE_RUN,goto $(V))

migrate-version:                             ## Migrations | Show current migration version
	@$(call _MIGRATE_RUN,version)

migrate-force:                               ## Migrations | Force version to V (clears dirty), e.g., V=1
	@if [ -z "$(V)" ]; then echo "Usage: make migrate-force V=1"; exit 2; fi
	@$(call _MIGRATE_RUN,force $(V))

migrate-new:                                 ## Migrations | Create new pair: 000X_NAME.(up or down).sql (NAME=xxx)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-new NAME=add_users_debug_note"; exit 2; fi
	@mkdir -p "$(MIGRATIONS_DIR)"
	@last=$$(ls -1 "$(MIGRATIONS_DIR)"/*_*.up.sql 2>/dev/null | sed -E 's#.*/([0-9]{4}).*#\1#' | sort -n | tail -1); \
	v=$$([ -z "$$last" ] && printf "0001" || printf "%04d" $$(($$last+1))); \
	up="$(MIGRATIONS_DIR)/$${v}_$(NAME).up.sql"; \
	down="$(MIGRATIONS_DIR)/$${v}_$(NAME).down.sql"; \
	printf "BEGIN;\n-- TODO: write migration\nCOMMIT;\n" > "$$up"; \
	printf "BEGIN;\n-- TODO: write rollback\nCOMMIT;\n" > "$$down"; \
	echo "Created: $$up and $$down"