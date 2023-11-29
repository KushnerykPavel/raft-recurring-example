start cluster command:

```shell
$ SERVER_PORT=2221 RAFT_NODE_ID=node1 RAFT_PORT=1111 RAFT_VOL_DIR=node_1_data go run cmd/main.go
$ SERVER_PORT=2222 RAFT_NODE_ID=node2 RAFT_PORT=1112 RAFT_VOL_DIR=node_2_data go run cmd/main.go
$ SERVER_PORT=2223 RAFT_NODE_ID=node3 RAFT_PORT=1113 RAFT_VOL_DIR=node_3_data go run cmd/main.go
```

start frontend command:

```shell
$ cd frontend && go run main.go
```