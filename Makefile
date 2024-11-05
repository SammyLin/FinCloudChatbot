.PHONY: run dev clean build ngrok

# 基本變量
BINARY_NAME=line-bot-server
PORT=4000

build:
	@echo "Building..."
	go build -o ${BINARY_NAME} main.go

run: build
	@echo "Running server..."
	./${BINARY_NAME}

dev:
	@echo "Running in development mode..."
	DEBUG_LOGGING=true go run main.go

ngrok:
	@echo "Starting ngrok tunnel..."
	ngrok http ${PORT}

clean:
	@echo "Cleaning..."
	go clean
	rm -f ${BINARY_NAME}

# 一次性啟動所有服務
up:
	@echo "Starting all services..."
	@make build
	@echo "Starting ngrok in background..."
	@ngrok http ${PORT} > ngrok.log 2>&1 &
	@echo "Waiting for ngrok to start..."
	@sleep 2
	@echo "Ngrok URL:"
	@curl -s http://localhost:4040/api/tunnels | grep -o '"public_url":"[^"]*' | grep -o 'https://.*'
	@echo "\nStarting server..."
	@DEBUG_LOGGING=true ./${BINARY_NAME} 