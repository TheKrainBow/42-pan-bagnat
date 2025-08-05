#########################################################################################
#                                       CONFIG                                          #
#########################################################################################
NETWORK := pan-bagnat-net
DOCKER_COMPOSE := docker compose
DATABASE_URL := postgres://admin:pw_admin@localhost/panbagnat?sslmode=disable

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
local-front:																			## Local | Stop frontend docker container, and run front locally (dev mode)
# 	$(DOCKER_COMPOSE) stop frontend  ## Not needed since local frontend hit on a different port
	cd frontend && BROWSER=none pnpm dev

local-back:																				## Local | Stop backend docker container, and run back locally (dev mode)
	$(DOCKER_COMPOSE) stop backend 
	cd backend/srcs && DATABASE_URL=$(DATABASE_URL) go run main.go

#########################################################################################
#                                      DATABASE                                         #
#########################################################################################
.PHONY: db-clear db-test db-clear-data db-init-schema
db-prune:																				## Database | Prune database datas, schemas and templates
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	docker exec pan-bagnat-db-1 bash -lc "dropdb -U admin --if-exists schema_template"

db-init-schema: db-prune																## Database | Push schema to database
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat < ./db/init/01_init.sql
	docker exec -i pan-bagnat-db-1 \
	  bash -lc "/docker-entrypoint-initdb.d/02_make_template.sh"

db-clear-data:																			## Database | Clear database datas
	docker exec -i pan-bagnat-db-1 \
	  psql -U admin -d panbagnat -c "\
	    TRUNCATE module_roles, user_roles, modules, roles, users RESTART IDENTITY CASCADE;\
	  "

db-test: db-clear-data																	## Database | Set database datas with test datas
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat < ./db/test_data.sql

#########################################################################################
#                                       DOCKER                                          #
#########################################################################################
.PHONY: up up-dev down build build-back build-front prune fprune
up:																						## Docker | Up latest built images for all containers. (Doesn't rebuild using your local files)
	@echo "⏳ Ensuring network '$(NETWORK)' exists…"
	@if ! docker network ls --format '{{.Name}}' | grep -q "^$(NETWORK)$$" ; then \
		echo "✅ Creating network '$(NETWORK)'"; \
		docker network create --driver bridge $(NETWORK); \
	else \
		echo "✅ Network '$(NETWORK)' already exists"; \
	fi
	@echo "🚀 Bringing up services…"
	$(DOCKER_COMPOSE) up -d

down:																					## Docker | Down docker images. (Doesn't delete images)
	@for dir in repos/*; do \
		if [ -d $$dir ] && [ -f $$dir/docker-compose.yml ]; then \
			echo "==> Stopping containers in $$dir"; \
			(cd $$dir && docker compose stop); \
		fi \
	done
	$(DOCKER_COMPOSE) down

prune:																					## Docker | Delete created images
	@echo "🛑 Bringing down containers…"
	$(DOCKER_COMPOSE) down
	@echo "🗑  Pruning images…"
	docker image prune -f
	@echo "🔍 Checking network '$(NETWORK)' usage…"
	@if [ "$$(docker network inspect $(NETWORK) --format '{{len .Containers}}')" -eq "0" ]; then \
		echo "🗑  No containers attached—removing network '$(NETWORK)'"; \
		docker network rm $(NETWORK); \
	else \
		echo "ℹ️  Network '$(NETWORK)' still in use—skipping removal"; \
	fi

build: 																					## Docker | Build all images and replace currently running images
	$(DOCKER_COMPOSE) build
	@$(MAKE) -s up

build-back: 																			## Docker | Build backend image and replace currently running backend image
	$(DOCKER_COMPOSE) build backend
	@$(MAKE) -s up

build-front: 																			## Docker | Build frontend image and replace currently running frontend image
	$(DOCKER_COMPOSE) build frontend
	@$(MAKE) -s up

fprune: prune																			## Docker | Stop all containers, volumes, and networks
	$(DOCKER_COMPOSE) down --volumes --remove-orphans || true
	docker network rm pan-bagnat_default 2>/dev/null || true
	docker system prune -af --volumes || true

REPO_DIRS := $(wildcard repos/*)

.PHONY: fprune-all
fprune-all:
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
	cd backend/srcs && DATABASE_URL=$(DATABASE_URL) go test -timeout 30s ./...

test-backend-verbose: 																	## Tests | Start tests for backend with verbose enabled
	@echo "🧪 Running backend tests…"
	cd backend/srcs && DATABASE_URL=$(DATABASE_URL) go test -v -timeout 30s ./...

#########################################################################################
#                                      DATABASE                                         #
#########################################################################################
swagger:
	cd backend/srcs && swag init -g main.go