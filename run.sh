#!/bin/bash

N=10
BASE_PORT=8080

start() {
    echo "Starting $N nodes..."
    pids=()
    for i in $(seq 1 $N); do
        port=$((BASE_PORT + i))
        id="node$i"
        peers=""
        for j in $(seq 1 $N); do
            if [ $i -ne $j ]; then
                peer_port=$((BASE_PORT + j))
                if [ -z "$peers" ]; then
                    peers="localhost:$peer_port"
                else
                    peers="$peers,localhost:$peer_port"
                fi
            fi
        done

        # --- THIS IS THE ONLY LINE THAT CHANGES ---
        go run ./cmd/server -id "$id" -addr "localhost:$port" -peers "$peers" &

        pid=$!
        pids+=($pid)
        echo "Started $id on port $port with PID $pid"
    done

    echo "${pids[@]}" > nodes.pid
    echo "All nodes started."
}

stop() {
    if [ -f "nodes.pid" ]; then
        pids=$(cat nodes.pid)
        echo "Stopping nodes with PIDs: $pids"
        kill $pids
        rm nodes.pid
        echo "All nodes stopped."
    else
        echo "PID file not found. Are the nodes running?"
    fi
}

case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    *)
        echo "Usage: $0 {start|stop}"
        exit 1
        ;;
esac