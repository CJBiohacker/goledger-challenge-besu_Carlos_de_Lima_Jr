#!/bin/bash

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../colors.sh"

BESU=./bin/besu

mkdir tmp && cd tmp
../$BESU operator generate-blockchain-config --config-file=../minimal/config.json --to=network --private-key-file-name=key

cd ..

counter=1
for folder in tmp/network/keys/*; do
    mkdir -p "nodes/node-$counter/data"
    cp -r "$folder"/* "nodes/node-$counter/data"
    ((counter++))
done

mkdir -p genesis
cp tmp/network/genesis.json genesis/genesis.json

if [ "$OS" = "Darwin" ]; then
    rm -rf tmp
else
    sudo rm -rf tmp
fi
echo

echo -e "${BLUE}Starting docker network 'besu_test_network'...${NC}"
docker network create --driver bridge besu_test_network
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Docker network created successfully.${NC}\n"
else
    echo -e "${YELLOW}Docker network may already exist. Continuing...${NC}\n"
fi

echo -e "${BLUE}Starting besu.node-1 on docker...${NC}"
docker run -d \
    --name besu.node-1 \
    --user root \
    -v "$(pwd)/nodes/node-1/data:/opt/besu/data" \
    -v "$(pwd)/genesis:/opt/besu/genesis" \
    -v "$(pwd)/minimal/config.toml:/opt/besu/config.toml" \
    -p 30303:30303 \
    -p 8545:8545 \
    -p 8546:8546 \
    -p 30303:30303/udp \
    -p 8545:8545/udp \
    --network besu_test_network \
    --restart always \
    hyperledger/besu:25.4.1 \
    --config-file=/opt/besu/config.toml

echo

echo -e "${BLUE}Waiting for besu.node-1 to be responsive...${NC}"
until curl -s -X POST --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8545/ > /dev/null 2>&1; do
    printf '.'
    sleep 2
done
echo -e "\n${GREEN}besu.node-1 is responsive!${NC}\n"

# ENODE_RESPONSE=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"net_enode","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8545/)
# ENODE_URL=$(echo $ENODE_RESPONSE | jq -r '.result')
# echo $ENODE_URL > minimal/bootnodes.txt

echo -e "${BLUE}Waiting for P2P subsystem to generate Enode...${NC}"
ENODE_URL="null"

# Loops until Besu stops erroring and returns the real URL
until [ "$ENODE_URL" != "null" ] && [ -n "$ENODE_URL" ]; do
    ENODE_RESPONSE=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"net_enode","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8545/ 2>/dev/null)
    ENODE_URL=$(echo $ENODE_RESPONSE | jq -r '.result')
    
    if [ "$ENODE_URL" == "null" ]; then
        printf '*'
        sleep 2
    fi
done

echo -e "\n${GREEN}Enode captured successfully!${NC}"
echo $ENODE_URL > minimal/bootnodes.txt

HOST_IP=$(docker container inspect besu.node-1 | jq -r '.[0].NetworkSettings.Networks.besu_test_network.IPAddress')
sed -i.bak "s/127.0.0.1/$HOST_IP/g" minimal/bootnodes.txt

echo -e "${BLUE}Starting besu.node-2 on docker...${NC}"
docker run -d \
    --name besu.node-2 \
    --user root \
    -v "$(pwd)/nodes/node-2/data:/opt/besu/data" \
    -v "$(pwd)/genesis:/opt/besu/genesis" \
    -v "$(pwd)/minimal/config.toml:/opt/besu/config.toml" \
    -v "$(pwd)/minimal/bootnodes.txt:/opt/besu/bootnodes.txt" \
    -p 30304:30303 \
    -p 8547:8545 \
    -p 8548:8546 \
    -p 30304:30303/udp \
    -p 8547:8545/udp \
    --network besu_test_network \
    --restart always \
    hyperledger/besu:25.4.1 \
    --config-file=/opt/besu/config.toml --bootnodes=$(cat ./minimal/bootnodes.txt)

echo

echo -e "${BLUE}Waiting for besu.node-2 to be responsive...${NC}"
until curl -s -X POST --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8547/ > /dev/null 2>&1; do
    printf '.'
    sleep 2
done
echo -e "\n${GREEN}besu.node-2 is responsive!${NC}\n"

echo -e "${BLUE}Starting besu.node-3 on docker...${NC}"
docker run -d \
    --name besu.node-3 \
    --user root \
    -v "$(pwd)/nodes/node-3/data:/opt/besu/data" \
    -v "$(pwd)/genesis:/opt/besu/genesis" \
    -v "$(pwd)/minimal/config.toml:/opt/besu/config.toml" \
    -v "$(pwd)/minimal/bootnodes.txt:/opt/besu/bootnodes.txt" \
    -p 30305:30303 \
    -p 8549:8545 \
    -p 8550:8546 \
    -p 30305:30303/udp \
    -p 8549:8545/udp \
    --network besu_test_network \
    --restart always \
    hyperledger/besu:25.4.1 \
    --config-file=/opt/besu/config.toml --bootnodes=$(cat ./minimal/bootnodes.txt) 

echo

echo -e "${BLUE}Waiting for besu.node-3 to be responsive...${NC}"
until curl -s -X POST --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8549/ > /dev/null 2>&1; do
    printf '.'
    sleep 2
done
echo -e "\n${GREEN}besu.node-3 is responsive!${NC}\n"

echo -e "${BLUE}Starting besu.node-4 on docker...${NC}"
docker run -d \
    --name besu.node-4 \
    --user root \
    -v "$(pwd)/nodes/node-4/data:/opt/besu/data" \
    -v "$(pwd)/genesis:/opt/besu/genesis" \
    -v "$(pwd)/minimal/config.toml:/opt/besu/config.toml" \
    -v "$(pwd)/minimal/bootnodes.txt:/opt/besu/bootnodes.txt" \
    -p 30306:30303 \
    -p 8551:8545 \
    -p 8552:8546 \
    -p 30306:30303/udp \
    -p 8551:8545/udp \
    --network besu_test_network \
    --restart always \
    hyperledger/besu:25.4.1 \
    --config-file=/opt/besu/config.toml --bootnodes=$(cat ./minimal/bootnodes.txt)

echo

echo -e "${BLUE}Waiting for besu.node-4 to be responsive...${NC}"
until curl -s -X POST --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' -H "Content-Type: application/json" http://localhost:8551/ > /dev/null 2>&1; do
    printf '.'
    sleep 2
done
echo -e "\n${GREEN}besu.node-4 is responsive!${NC}\n"

echo -e "${GREEN}============================="
echo -e "Network started successfully!"
echo -e "=============================${NC}\n"