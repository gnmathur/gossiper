# Gossiper - Distributed Game State Synchronization Server

A lightweight distributed game server that uses gossip protocol to synchronize player states across multiple nodes. This system enables eventual consistency of player scores across a cluster of game servers without requiring a centralized database.

## Design Choices and Trade-offs

### Architecture Decisions
- **Gossip Protocol**: Chosen for its simplicity and resilience. Each node periodically exchanges state with random peers, ensuring eventual consistency without complex coordination.
- **Pull-based Synchronization**: Nodes actively pull state from peers every 2 seconds, providing predictable network usage patterns.
- **Last-Write-Wins (LWW)**: Conflict resolution uses timestamp-based LWW strategy, favoring simplicity over complex conflict resolution.
- **In-Memory Storage**: Player states are stored in memory for fast access, trading durability for performance.

### Trade-offs Made
- **Eventual Consistency**: Accepts temporary inconsistencies for better availability and partition tolerance (AP in CAP theorem).
- **No Persistence**: Data is lost on restart, suitable for session-based gaming but not for permanent player progress.
- **Simple Conflict Resolution**: LWW may lose updates in concurrent scenarios but keeps the implementation straightforward.

## Usage

### Prerequisites
- Go 1.24.0 or higher
- Network connectivity between nodes for multi-node setup

### Building the Server
```bash
go build -o gossiper ./cmd/server
```

### Running a Single Node
```bash
./gossiper --id=node1 --addr=localhost:8081
```

### Running Multiple Nodes (Cluster)
Start three nodes that gossip with each other:

**Node 1:**
```bash
./gossiper --id=node1 --addr=localhost:8081 --peers=localhost:8082,localhost:8083
```

**Node 2:**
```bash
./gossiper --id=node2 --addr=localhost:8082 --peers=localhost:8081,localhost:8083
```

**Node 3:**
```bash
./gossiper --id=node3 --addr=localhost:8083 --peers=localhost:8081,localhost:8082
```

### Command-Line Options

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `--id` | Unique identifier for the node | `node1` | `--id=game-server-1` |
| `--addr` | HTTP server address (host:port) | `localhost:8081` | `--addr=0.0.0.0:8080` |
| `--peers` | Comma-separated list of peer addresses | `""` (empty) | `--peers=10.0.0.2:8080,10.0.0.3:8080` |

### API Endpoints

#### Update Player Score
Updates a player's score on the local node. The update will propagate to other nodes via gossip.

```bash
curl "http://localhost:8081/update?playerId=player123&score=1500"
```

**Parameters:**
- `playerId`: Unique identifier for the player (required)
- `score`: Player's new score (required, integer)

#### Get Current State
Retrieves the current state of all players known to this node.

```bash
curl "http://localhost:8081/state"
```

**Response Example:**
```json
{
  "player123": {
    "score": 1500,
    "timestamp": 1696012345000000000
  },
  "player456": {
    "score": 2300,
    "timestamp": 1696012346000000000
  }
}
```

#### Gossip Endpoint (Internal)
Used internally by nodes to exchange state. Not intended for direct client use.

```
POST /gossip
Content-Type: application/json
```

### Web Interface
Access the web interface by navigating to the server's address in a browser:
```
http://localhost:8081
```

## Implementation Notes

### Gossip Mechanism
- Each node maintains a map of player IDs to their states (score and timestamp)
- Every 2 seconds, each node randomly selects a peer and sends its complete state
- Upon receiving state from a peer, the node merges it with its local state
- Conflicts are resolved using timestamps (Last-Write-Wins)

### Concurrency Safety
- All state mutations are protected by read-write mutexes
- Gossip operations create deep copies to prevent data races
- HTTP handlers are safe for concurrent requests

### Network Resilience
- Failed gossip attempts are silently ignored (nodes may be temporarily unavailable)
- System continues to function with partial network connectivity
- Nodes automatically recover when connectivity is restored

### Scalability Considerations
- Gossip interval can be adjusted based on cluster size and network capacity
- Full state transfer may become inefficient with large player counts
- Consider implementing delta-based gossip for production use

## Example Use Cases

### Local Development Testing
Test state synchronization between nodes:

1. Start multiple nodes as shown above
2. Update a player's score on node1:
   ```bash
   curl "http://localhost:8081/update?playerId=alice&score=100"
   ```
3. Wait a few seconds for gossip propagation
4. Check the state on node2:
   ```bash
   curl "http://localhost:8082/state"
   ```
   You should see alice's score reflected on all nodes.

### Load Distribution
Deploy multiple gossiper instances behind a load balancer. Players can connect to any node, and their states will eventually synchronize across all nodes.

### Geographic Distribution
Deploy nodes in different regions. Players connect to their nearest node for low latency, while gossip ensures global state consistency.

## Limitations

- **Data Persistence**: No built-in persistence; all data is lost on restart
- **Security**: No authentication or encryption for API endpoints or gossip communication
- **Message Ordering**: No guarantee of causal ordering for updates
- **Network Overhead**: Full state transfer can be inefficient for large player bases
- **Conflict Resolution**: Simple LWW may lose updates in high-concurrency scenarios

## Future Improvements

- Add persistent storage backend (e.g., RocksDB, BadgerDB)
- Implement delta-based gossip to reduce network overhead
- Add authentication and TLS for secure communication
- Implement vector clocks for better conflict resolution
- Add metrics and monitoring endpoints
- Support for player metadata beyond scores
- Configurable gossip intervals and fanout