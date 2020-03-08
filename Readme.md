# Flocons

Distributed file system to handle billions of small files

## Installation

```
go get -v -u github.com/t-mind/flocons
go run github.com/t-mind/flocons/main --config <config-file>
```

Examples of config file can be found in \$GOPATH/rc/github.com/t-mind/flocons/resources

## Test your application

### List all files in a directory

`curl http://localhost:<port>/files/<directory-path>`

### Read a file

`curl http://localhost:<port>/files/<file-path>`

### Create a directory

`curl -X POST -H "Content-Type:inode/directory" http://localhost:<port>/files/<directory-path>`

### Create a file

`curl -d <content> -H "Content-Type:<content-type>" http://localhost:<port>/files/<file-path>`

Or, if you have a file to upload

`curl "--data=@<path-to-local-file>" -H "Content-Type:<content-type>" http://localhost:<port>/files/<file-path>`

## Configuration description

```
{
  "namespace": "namespace to separate multiple flocons cluster using the same zookeeper. default is flocons",
  "zookeeper": ["list of zookeeper addresses"],
  "node": {
    "name": "uniquely identifies the node",
    "port": "port for the http server. If not set, a random port will be used",
    "external_address": "address for other nodes to communicate",
    "shard": "shard name"
  },
  "storage": {
    "path": "where the files will be stored on the local system",
    "max_size": "max total size of the storage in format '1GB'",
    "max_container_size": "max size of one container inside a directory. Default is 100MB"
  }
}
```

## To complete

| Feature | Included | Implemented | Comment |
| ------- | -------- | ----------- | ------- |


## Components

### Storage

### Cluster topoly client

### Dispatcher

Needs

- Cluster topology client

### Http server

Needs:

- Storage
- Cluster topology client
- Dispatcher

### Http file client

Needs

- Cluster topology client

### Fuse server

Needs

- Http file client
- Cluster topology client

### Replication manager

Needs

- Storage
- Cluster topology client
