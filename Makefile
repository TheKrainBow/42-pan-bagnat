#########################################################################################
#                                       CONFIG                                          #
#########################################################################################
DOCKER_COMPOSE = docker compose
DATABASE_URL = postgres://admin:pw_admin@localhost/panbagnat?sslmode=disable

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
	$(DOCKER_COMPOSE) stop frontend
	cd frontend && BROWSER=none pnpm start

local-back:																				## Local | Stop backend docker container, and run back locally (dev mode)
	$(DOCKER_COMPOSE) stop backend 
	cd backend/srcs && DATABASE_URL=$(DATABASE_URL) go run main.go

#########################################################################################
#                                      DATABASE                                         #
#########################################################################################
.PHONY: db-clear db-test
db-clear:																				## Database | Clear database datas and schema
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

db-test: db-clear																		## Database | Clear database and populate it with some test datas
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat < ./db/init.sql
	docker exec -i pan-bagnat-db-1 psql -U admin -d panbagnat < ./db/test_data.sql

#########################################################################################
#                                       DOCKER                                          #
#########################################################################################
.PHONY: up down build build-back build-front prune fprune
up:																						## Docker | Up latest built images. (Doesn't rebuild using your local files)
	$(DOCKER_COMPOSE) up -d

down:																					## Docker | Down docker images. (Doesn't delete images)
	$(DOCKER_COMPOSE) down

prune:																					## Docker | Delete created images
	$(DOCKER_COMPOSE) down
	docker image prune -f

build: 																					## Docker | Build all images and replace currently running images
	$(DOCKER_COMPOSE) build
	$(DOCKER_COMPOSE) up -d

build-back: 																			## Docker | Build backend image and replace currently running backend image
	$(DOCKER_COMPOSE) build backend
	$(DOCKER_COMPOSE) up -d

build-front: 																			## Docker | Build frontend image and replace currently running frontend image
	$(DOCKER_COMPOSE) build frontend
	$(DOCKER_COMPOSE) up -d

fprune: prune																			## Docker | Stop all containers, volumes, and networks
	$(DOCKER_COMPOSE) down --volumes --remove-orphans
	docker system prune -af --volumes
