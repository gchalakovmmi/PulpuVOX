NAME := pulpuvox
DB_USR := changeme
DB_SCH := app

.PHONY: clear backend backend-logs database database-clean database-logs database-connect objectstorage objectstorage-logs all up down restart

clear:
	@clear

backend:
	@echo "=== Backend ==="
	@echo "Generating templ files..."
	@cd backend/app && make generate
	@echo "Compiling source code..."
	@cd backend/app && make build
	@echo "Building image..."
	@cd backend && docker buildx build -t $(NAME)-backend .
	@echo "Recreating container..."
	@docker compose up -d --no-deps --force-recreate backend
	@docker inspect -f '{{.NetworkSettings.Networks.net.IPAddress}}' $(NAME)-backend-1

backend-logs:
	@docker logs --follow $(NAME)-backend-1

database-clean:
	@docker stop $(NAME)-database-1
	@docker rm $(NAME)-database-1
	@docker volume rm $(NAME)_database_data

database:
	@echo "=== DataBase ==="
	@echo "Building image..."
	@cd database && docker buildx build -t $(NAME)-postgres .
	@echo "Recreating container..."
	@docker compose up -d --no-deps --force-recreate database
	@docker inspect -f '{{.NetworkSettings.Networks.net.IPAddress}}' $(NAME)-database-1

database-logs:
	@docker logs --follow $(NAME)-database-1

database-connect:
	@echo "Waiting for PostgreSQL to be ready..."
	@timeout 30 bash -c 'until docker exec $(NAME)-database-1 pg_isready -U $(DB_USR) -d $(DB_SCH); do sleep 0.5; done'
	@sleep 0.5
	@docker exec -it $(NAME)-database-1 psql -h localhost -U $(DB_USR) -d $(DB_SCH)

objectstorage:
	@echo "=== ObjectStorage ==="
	@echo "Building image..."
	@cd objectstorage && docker buildx build -t $(NAME)-objectstorage .
	@echo "Recreating container..."
	@docker compose up -d --no-deps --force-recreate objectstorage
	@docker inspect -f '{{.NetworkSettings.Networks.net.IPAddress}}' $(NAME)-objectstorage-1

objectstorage-logs:
	@docker logs --follow $(NAME)-objectstorage-1

all: backend database objectstorage

up:
	docker compose up -d

down:
	docker compose down

restart: down up
