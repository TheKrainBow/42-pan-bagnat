# Makefile for handling Docker Compose operations

.PHONY: up build down prune fprune

DOCKER_COMPOSE = docker compose

build:
	$(DOCKER_COMPOSE) build
	$(DOCKER_COMPOSE) up -d

build-back:
	$(DOCKER_COMPOSE) build backend
	$(DOCKER_COMPOSE) up -d

build-front:
	$(DOCKER_COMPOSE) build frontend
	$(DOCKER_COMPOSE) up -d

local-front:
	$(DOCKER_COMPOSE) stop frontend
	cd frontend && BROWSER=none pnpm start

local-back:
	$(DOCKER_COMPOSE) stop backend 
	cd backend && go run main.go

up:
	$(DOCKER_COMPOSE) up -d

down:
	$(DOCKER_COMPOSE) down

prune:
	docker image prune -f

fprune:
	$(DOCKER_COMPOSE) down --volumes --remove-orphans
	docker system prune -af --volumes
