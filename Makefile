.PHONY: all build-cli test test-docker start-soc stop-soc

all: build-cli

build-cli:
	@echo "🔨 Building agentsh Go CLI..."
	cd cli && go build -o agentsh main.go && ln -sf agentsh ash && ln -sf agentsh sd

test:
	@echo "🧪 Running unit tests in containerized test harness..."
	./scripts/test-in-container.sh

start-soc:
	@echo "🐳 Starting the agentsh Control Plane + Dashboard via Docker Compose..."
	docker compose up -d

stop-soc:
	@echo "🛑 Stopping the agentsh SOC..."
	docker compose down
