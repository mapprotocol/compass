# Compass

Compass is the Golang implementation of cross-chain communication maintainer for MAP Protocol. It currently supports bridging between EVM based chains.

The Compass is an independent service, it contains two operating modes, [Maintainer](#maintainer) and [Messenger](#messenger) mode, users need specify a mode to start the service program

The newly designed compass version contains all the functions required to run the relay node. With this tool, you can run nodes on almost all hardware platforms.

This project is inspired by [ChainSafe/ChainBridge](https://github.com/ChainSafe/ChainBridge)

# Contents

- [Compass](#compass)
- [Contents](#contents)
- [Quick Start](#quick-start)
    - [2. Prepare the accounts for each chain](#2-prepare-the-accounts-for-each-chain)
    - [3. Modify the configuration file](#3-modify-the-configuration-file)
    - [4. Running the executable](#4-running-the-executable)
- [Building](#building)
- [Maintainer](#maintainer)
- [Messenger](#messenger)
- [Monitor](#monitor)
- [Configuration](#configuration)
    - [Options](#options)
  - [Blockstore](#blockstore)
  - [Keystore](#keystore)
- [Chain Implementations](#chain-implementations)
  - [Near](#near)

# Quick Start

the recommanded way to get the executable is to download it from the release page.

>if you want to build it from the source code,check the [building](#building) section below.

### 2. Prepare the accounts for each chain
fund some accounts in order to send txs on each chain, you want to provice crosse-chain service.
the esaiest way is to using the same one address for every chain.

after that we need to import the account into the keystore of compass.  
using the private key is the simplest way,run the following command in terminal:

```zsh
compass accounts import --privateKey '********** your private key **********'
```

during the process of importing, you will be asked to input a password.  
the password is used to encrypt your keystore.you have to input it when unlocking your account.

to list the imported keys in the keystore, using the command below:
```zsh
compass accounts list
```

### 3. Modify the configuration file
copy a example configure file from
```json
{
  "mapchain": {
    "id": "212",
    "endpoint": "http://18.142.54.137:7445",
    "from": "0xE0DC8D7f134d0A79019BEF9C2fd4b2013a64fCD6",
    "opts": {
      "mcs": "0x0ac4611305254cdd257beC56CB79CBeC720Cd02D",
      "lightnode": "0x000068656164657273746F726541646472657373",
      "http": "true",
      "gasLimit": "4000000000000",
      "maxGasPrice": "2000000000000",
      "syncIdList": "[34434]"
    }
  },
  "chains": [
    {
      "name": "pri-eth",
      "type": "ethereum",
      "id": "34434",
      "endpoint": "http://18.138.248.113:8545",
      "from": "0xE0DC8D7f134d0A79019BEF9C2fd4b2013a64fCD6",
      "opts": {
        "mcs": "0xcfc80beddb70f12af6da768fc30e396889dfce26",
        "lightnode": "0x80Be41aEBFdaDBD58a65aa549cB266dAFb6b8304",
        "http": "true",
        "gasLimit": "400000000000",
        "maxGasPrice": "200000000000",
        "syncToMap": "true"
      }
    }
  ]
}
```
modify the configuration accordingly.  
fill the accounts for each chain.

### 4. Running the executable
lauch and keep the executable runing simply by run:
```zsh
compass maintainer --blockstore ./block-eth-map --config ./config-mcs-erh-map.json
```
you will be asked to input the password to unlock your account.(which you have inputed at step 2)
if everything runs smoothly. it's all set

# Building

Building compass requires a [Go](https://github.com/golang/go) compiler(version 1.16 or later)

under the root directory of the repo

`make build`: Builds `compass` in `./build`.  
`make install`: Uses `go install` to add `compass` to your GOBIN.

# Maintainer

Synchronize the information of blocks in each chain according to the information in the configuration file

Start with the following command:
```zsh
compass maintainer --blockstore ./block-eth-map --config ./config.json
```

# Messenger

Synchronize the log information of transactions of blocks in each chain according to the information in the configuration file

Start with the following command:
```zsh
compass messenger --blockstore ./block-eth-map --config ./config.json
```

# Monitor

Initiate monitoring of user balances and transactions

# Configuration

the configuration file is a small JSON file.

```
{
  "mapchain": {
        "id": "0",                          // Chain ID of the MAP chain
        "endpoint": "ws://<host>:<port>",   // Node endpoint
        "from": "0xff93...",                // MAP chain address of maintainer
        "opts": {}                          // MAP Chain configuration options (see below)
    },
  "chains": []                              // List of Chain configurations
}

```

A chain configurations take this form:

```
{
    "name": "eth",                      // Human-readable name
    "type": "ethereum",                 // Chain type (Please see the following cousin for details)
    "id": "0",                          // Chain ID
    "endpoint": "ws://<host>:<port>",   // Node endpoint
    "from": "0xff93...",                // On-chain address of maintainer
    "keystorePath" : "/you/path/",      // 
    "opts": {},                         // Chain-specific configuration options (see below)
}
```

|  chain   | type     |
|:--------:|----------|
| ethereum | ethereum |
|   bsc    | bsc      |
|  goerli  | eth2     |
| polygon  | matic    |
|   near   | near     |
|  klaytn  | klaytn   |
|  platon  | platon   |

See `config.json.example` for an example configuration.

### Options

Since MAP is also a EVM based chain, so the opts of the **mapchain** is following the options below as well  
Ethereum chains support the following additional options:

```
{
    "mcs": "0x12345...",                                    // Address of the bridge contract (required)
    "maxGasPrice": "0x1234",                                // Gas price for transactions (default: 20000000000)
    "gasLimit": "0x1234",                                   // Gas limit for transactions (default: 6721975)
    "gasMultiplier": "1.25",                                // Multiplies the gas price by the supplied value (default: 1)
    "http": "true",                                         // Whether the chain connection is ws or http (default: false)
    "startBlock": "1234",                                   // The block to start processing events from (default: 0)
    "blockConfirmations": "10"                              // Number of blocks to wait before processing a block
    "egsApiKey": "xxx..."                                   // API key for Eth Gas Station (https://www.ethgasstation.info/)
    "egsSpeed": "fast"                                      // Desired speed for gas price selection, the options are: "average", "fast", "fastest"
    "lightnode": "0x12345...",                              // the lightnode to sync header
    "syncToMap": "true",                                    // Whether sync blockchain headers to Map
    "syncIdList": "[214]"                                   // Those chain ids are synchronized to the map，and This configuration can only be used in mapchain
    "event": "mapTransferOut(...)|depositOutToken(...)",    // MCS events monitored by the program, multiple with | interval，
                                                            // Here we give the events that need to be monitored，Map:mapTransferOut(bytes,bytes,bytes32,uint256,uint256,bytes,uint256,bytes) Near: 2ef1cdf83614a69568ed2c96a275dd7fb2e63a464aa3a0ffe79f55d538c8b3b5|150bd848adaf4e3e699dcac82d75f111c078ce893375373593cc1b9208998377
    "waterLine": "5000000000000000000",                     // If the user balance is lower than, an alarm will be triggered, unit ：wei
    "alarmSecond": "3000",                                  // How long does the user balance remain unchanged, triggering the alarm, unit ：seconds                                              
}
```
## Blockstore

The blockstore is used to record the last block the maintainer processed, so it can pick up where it left off.

To disable loading from the chunk library, specify the "--fresh" flag. Add the fresh flag, and the program will execute from height 0，

In addition, the configuration file provides the "startBlock" option, and the program will execute from the startBlock

## Keystore

Compass requires keys to sign and submit transactions, and to identify each bridge node on chain.

To use secure keys, see `compass accounts --help`. The keystore password can be supplied with the `KEYSTORE_PASSWORD` environment variable.

To import external ethereum keys, such as those generated with geth, use `compass accounts import --ethereum /path/to/key`.

To import private keys as keystores, use `compass accounts import --privateKey key`.

# Chain Implementations

- Ethereum (Solidity): [contracts](https://github.com/mapprotocol/contracts)
  The Solidity contracts required for compass. Includes scripts for deployment.

## Near

If you need to synchronize the near block, please install the near cli first. Here is a simple tutorial. For more information, 

please check [Near cli installation tutorial](https://docs.near.org/tools/near-cli#installation)

First, install npm. Depending on the system, the running command is different. The following is an example of the installation command in 

the ubuntu system. Use the `apt install npm` command to run `npm install -g near cli`,

After installation, use 'near -- version' to check whether the installation is successful

Configure the environment you need, for example:

`
  export NEAR_CLI_LOCALNET_RPC_SERVER_URL=https://archival-rpc.testnet.near.org

  export NEAR_ENV=testnet
`

Use the 'near login' command , Creates a key pair locally in `.near-credentials` with an implicit account as the accountId. (hash representation of the public key)

And record the directory to the keystorePath option in the configuration file

In addition, another program needs to be run for near messenger. Please check [near-lake-s3](./near-lake-s3/README.md)
