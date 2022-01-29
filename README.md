# neofs-cngl
Simplified NeoFS data storage server

## Overview

Repository contains source code of the simplified NeoFS storage node.

The application is designed to be used for testing client applications 
which communicate with NeoFS using NeoFS API protocol.

## Build

```shell
$ make [build]
```

## Configuration

See `config/config.yaml`.

## Run

```shell
$ ./bin/neofs-cngl --config </path/to/config>
```