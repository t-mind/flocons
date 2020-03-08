# Flocons

Distributed file system to handle billions of small files

| Feature | Included | Implemented | Comment |
| ------- | -------- | ----------- | ------- |

## Components

### Storage

### Cluster topoly client

### Dispatcher

Needs

* Cluster topology client

### Http server

Needs:

* Storage
* Cluster topology client
* Dispatcher

### Http file client

Needs

* Cluster topology client

### Fuse server

Needs

* Http file client
* Cluster topology client

### Replication manager

Needs

* Storage
* Cluster topology client