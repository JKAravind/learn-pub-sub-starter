# ⚔️ Peril — Distributed Game Simulation (Event-Driven Architecture)

**Peril** is a distributed CLI-based strategy game inspired by the board game *Risk*.  
Each player runs as an independent Go service that communicates entirely through **RabbitMQ events** — no direct API calls.  

This project demonstrates the principles of **Event-Driven Architecture (EDA)**, **message brokering**, and **concurrency** in **Go**, powered by **RabbitMQ** and **Docker**.

---

## 🧩 Tech Stack
- **Go (Golang)** — Core logic and concurrency (goroutines, channels)
- **RabbitMQ** — Event broker for communication between services
- **Docker** — Portable setup for the RabbitMQ server
---

### 1️⃣ Prerequisites
Make sure you have the following installed:
- [Docker](https://docs.docker.com/get-docker/)
- [Go](https://go.dev/doc/install)
- Git (for cloning the repository)
---

### 2️⃣ Clone the Repository

### 3️⃣ Start RabbitMQ with Docker

Run the provided shell script to start RabbitMQ in a Docker container.

```bash
./rabbit.sh start
```

This script will:
- Pull the official RabbitMQ image (if not already available)
- Run it in a container
- Expose the necessary ports (default: `5672` for AMQP, `15672` for the management UI)

Once it’s running, you can access the RabbitMQ management UI at:
👉 [http://localhost:15672](http://localhost:15672)
---

### 4️⃣ Start the Game Server

The **server** is the core orchestrator that handles game state and broadcasts events.

```bash
go run ./cmd/server
```

The server supports actions like:
- **Pause**
- **Start**
- **Quit**

---

### 5️⃣ Start Player Clients

Each player acts as an independent Go service.
You can run multiple clients in separate terminals:

```bash
go run ./cmd/client
```

Each client listens to RabbitMQ events and performs actions like:
- Spawning armies
- Moving units
- Declaring wars

All interactions happen through RabbitMQ — no direct HTTP or API calls.

## 🧩 Example Flow

1. Server starts and listens for player events.
2. Clients join and send "spawn" or "move" events via RabbitMQ.
3. RabbitMQ routes messages between all players and the server.
4. The server updates game state and publishes new events.
5. All clients receive updates and react accordingly.

---

## 🧰 Future Improvements

- Web UI for visualizing the board state
- Game state persistence using a database
- Real-time dashboard with event logs

---
This project started as an experiment to explore **Event-Driven Architecture** and ended up teaching me how **distributed systems communicate** without chaining API calls.
If you're exploring **Go**, **RabbitMQ**, or **EDA**, this is a fun way to get your hands dirty with real message-driven systems!
---


**Aravind JK**  
