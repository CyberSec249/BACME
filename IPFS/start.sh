 #!/bin/bash

# 启动节点1
export IPFS_PATH=/home/acme/IPFS/node1
ipfs daemon &

export IPFS_PATH=/home/acme/IPFS/node2
ipfs daemon & 

export IPFS_PATH=/home/acme/IPFS/node3
ipfs daemon & 

export IPFS_PATH=/home/acme/IPFS/node4
ipfs daemon & 