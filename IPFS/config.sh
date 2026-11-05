#!/bin/bash

# 创建节点
mkdir node1
mkdir node2
mkdir node3
mkdir node4
cp swarm.key node1
cp swarm.key node2
cp swarm.key node3
cp swarm.key node4



# 初始化节点1
export IPFS_PATH=/home/acme/IPFS/node1
ipfs init
ipfs config --json Addresses.Swarm '["/ip4/0.0.0.0/tcp/4001", "/ip6/::/tcp/4001"]'
ipfs config Addresses.API "/ip4/0.0.0.0/tcp/9001"
ipfs config Addresses.Gateway "/ip4/0.0.0.0/tcp/8080"
ipfs bootstrap rm --all
NODE1_ADDR=$(ipfs id -f "<id>\n" | head -n 1)

# 初始化节点2
export IPFS_PATH=/home/acme/IPFS/node2
ipfs init
ipfs config --json Addresses.Swarm '["/ip4/0.0.0.0/tcp/4002", "/ip6/::/tcp/4002"]'
ipfs config Addresses.API "/ip4/0.0.0.0/tcp/9002"
ipfs config Addresses.Gateway "/ip4/0.0.0.0/tcp/8081"
ipfs bootstrap rm --all
NODE2_ADDR=$(ipfs id -f "<id>\n" | head -n 1)

# 初始化节点3
export IPFS_PATH=/home/acme/IPFS/node3
ipfs init
ipfs config --json Addresses.Swarm '["/ip4/0.0.0.0/tcp/4003", "/ip6/::/tcp/4003"]'
ipfs config Addresses.API "/ip4/0.0.0.0/tcp/9003"
ipfs config Addresses.Gateway "/ip4/0.0.0.0/tcp/8082"
ipfs bootstrap rm --all
NODE3_ADDR=$(ipfs id -f "<id>\n" | head -n 1)

# 初始化节点4
export IPFS_PATH=/home/acme/IPFS/node4
ipfs init
ipfs config --json Addresses.Swarm '["/ip4/0.0.0.0/tcp/4004", "/ip6/::/tcp/4004"]'
ipfs config Addresses.API "/ip4/0.0.0.0/tcp/9004"
ipfs config Addresses.Gateway "/ip4/0.0.0.0/tcp/8083"
ipfs bootstrap rm --all
NODE4_ADDR=$(ipfs id -f "<id>\n" | head -n 1)

# 添加节点
ipfs bootstrap add /ip4/127.0.0.1/tcp/4001/ipfs/$NODE1_ADDR
ipfs bootstrap add /ip4/127.0.0.1/tcp/4002/ipfs/$NODE2_ADDR
ipfs bootstrap add /ip4/127.0.0.1/tcp/4003/ipfs/$NODE3_ADDR
export IPFS_PATH=/home/acme/IPFS/node1
ipfs bootstrap add /ip4/127.0.0.1/tcp/4002/ipfs/$NODE2_ADDR
ipfs bootstrap add /ip4/127.0.0.1/tcp/4003/ipfs/$NODE3_ADDR
ipfs bootstrap add /ip4/127.0.0.1/tcp/4004/ipfs/$NODE4_ADDR
export IPFS_PATH=/home/acme/IPFS/node2
ipfs bootstrap add /ip4/127.0.0.1/tcp/4001/ipfs/$NODE1_ADDR
ipfs bootstrap add /ip4/127.0.0.1/tcp/4003/ipfs/$NODE3_ADDR
ipfs bootstrap add /ip4/127.0.0.1/tcp/4004/ipfs/$NODE4_ADDR
export IPFS_PATH=/home/acme/IPFS/node3
ipfs bootstrap add /ip4/127.0.0.1/tcp/4001/ipfs/$NODE1_ADDR
ipfs bootstrap add /ip4/127.0.0.1/tcp/4002/ipfs/$NODE2_ADDR
ipfs bootstrap add /ip4/127.0.0.1/tcp/4004/ipfs/$NODE4_ADDR