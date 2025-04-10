.PHONY: up build down prune fprune

DOCKER_COMPOSE = docker compose

build: # build all images and replace currently running images
	$(DOCKER_COMPOSE) build
	$(DOCKER_COMPOSE) up -d

build-back: # build backend image and replace currently running backend image
	$(DOCKER_COMPOSE) build backend
	$(DOCKER_COMPOSE) up -d

build-front: # build frontend image and replace currently running frontend image
	$(DOCKER_COMPOSE) build frontend
	$(DOCKER_COMPOSE) up -d

local-front: # stop frontend image, and run front locally (dev mode)
	$(DOCKER_COMPOSE) stop frontend
	cd frontend && BROWSER=none pnpm start

local-back: # stop backend image, and run back locally (dev mode)
	$(DOCKER_COMPOSE) stop backend 
	cd backend && go run main.go

up: # up latest built images. (Doesn't rebuild using your local files)
	$(DOCKER_COMPOSE) up -d

down: # down docker images. (Doesn't delete images)
	$(DOCKER_COMPOSE) down

prune: # Delete created images
	$(DOCKER_COMPOSE) down
	docker image prune -f

fprune: prune # Stop all containers, volumes, and networks
	$(DOCKER_COMPOSE) down --volumes --remove-orphans
	docker system prune -af --volumes

help: # Display help message with list of rules
	@echo "Makefile commands:"
	@grep -E '^[a-zA-Z0-9_-]+:' Makefile | sed 's/:.*#/ #/' | sed 's/#/-/'
