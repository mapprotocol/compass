// Copyright 2021 Compass Systems
// SPDX-License-Identifier: LGPL-3.0-only

package mapprotocol

const RelayerABIJSON = `[
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "from",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "value",
        "type": "uint256"
      }
    ],
    "name": "Register",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "from",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "value",
        "type": "uint256"
      }
    ],
    "name": "Unregister",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "from",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "value",
        "type": "uint256"
      }
    ],
    "name": "Withdraw",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "sender",
        "type": "address"
      },
      {
        "indexed": true,
        "internalType": "uint256",
        "name": "chainId",
        "type": "uint256"
      },
      {
        "indexed": true,
        "internalType": "bytes32",
        "name": "bindAddress",
        "type": "bytes32"
      }
    ],
    "name": "WorkerSet",
    "type": "event"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "addr",
        "type": "address"
      }
    ],
    "name": "address2Bytes",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "pure",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint256[]",
        "name": "_chainIdList",
        "type": "uint256[]"
      },
      {
        "internalType": "bytes32",
        "name": "_worker",
        "type": "bytes32"
      }
    ],
    "name": "batchBindingSingleWorker",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint256[]",
        "name": "_chainIdList",
        "type": "uint256[]"
      },
      {
        "internalType": "bytes32[]",
        "name": "_workerList",
        "type": "bytes32[]"
      }
    ],
    "name": "batchBindingWorkers",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "_worker",
        "type": "address"
      }
    ],
    "name": "bind",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint256",
        "name": "_chainId",
        "type": "uint256"
      },
      {
        "internalType": "bytes32",
        "name": "_worker",
        "type": "bytes32"
      }
    ],
    "name": "bindingWorker",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes32",
        "name": "b32",
        "type": "bytes32"
      }
    ],
    "name": "bytes2Address",
    "outputs": [
      {
        "internalType": "address",
        "name": "",
        "type": "address"
      }
    ],
    "stateMutability": "pure",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "uint256",
        "name": "chainID",
        "type": "uint256"
      }
    ],
    "name": "currentNumberAndHash",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "number",
        "type": "uint256"
      },
      {
        "internalType": "bytes",
        "name": "hash",
        "type": "bytes"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "length",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "register",
    "outputs": [],
    "stateMutability": "payable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "_relayer",
        "type": "address"
      }
    ],
    "name": "relayerAmount",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "_relayer",
        "type": "address"
      },
      {
        "internalType": "uint256",
        "name": "chainId",
        "type": "uint256"
      }
    ],
    "name": "relayerWorker",
    "outputs": [
      {
        "internalType": "bytes32",
        "name": "",
        "type": "bytes32"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "relayers",
    "outputs": [
      {
        "internalType": "address[]",
        "name": "",
        "type": "address[]"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
	   "inputs": [
		  {
			 "internalType": "bytes",
			 "name": "blockHeader",
			 "type": "bytes"
		  }
	   ],
	   "name": "updateBlockHeader",
	   "outputs": [],
	   "stateMutability": "nonpayable",
	   "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "router",
        "type": "address"
      },
      {
        "internalType": "address",
        "name": "coin",
        "type": "address"
      },
      {
        "internalType": "uint256",
        "name": "srcChain",
        "type": "uint256"
      },
      {
        "internalType": "uint256",
        "name": "dstChain",
        "type": "uint256"
      },
      {
        "internalType": "bytes",
        "name": "txProve",
        "type": "bytes"
      }
    ],
    "name": "txVerify",
    "outputs": [
      {
        "internalType": "bool",
        "name": "success",
        "type": "bool"
      },
      {
        "internalType": "string",
        "name": "message",
        "type": "string"
      }
    ],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "unregister",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "withdraw",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  }
]`

const LiteABIJSON = `[
  {
    "inputs": [],
    "stateMutability": "nonpayable",
    "type": "constructor"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": false,
        "internalType": "address",
        "name": "previousAdmin",
        "type": "address"
      },
      {
        "indexed": false,
        "internalType": "address",
        "name": "newAdmin",
        "type": "address"
      }
    ],
    "name": "AdminChanged",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "beacon",
        "type": "address"
      }
    ],
    "name": "BeaconUpgraded",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": true,
        "internalType": "address",
        "name": "implementation",
        "type": "address"
      }
    ],
    "name": "Upgraded",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {
        "indexed": false,
        "internalType": "uint256",
        "name": "epoch",
        "type": "uint256"
      }
    ],
    "name": "validitorsSet",
    "type": "event"
  },
  {
    "inputs": [],
    "name": "currentEpoch",
    "outputs": [
      {
        "internalType": "uint256",
        "name": "",
        "type": "uint256"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "currentValidators",
    "outputs": [
      {
        "internalType": "bytes[]",
        "name": "",
        "type": "bytes[]"
      }
    ],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes",
        "name": "firstBlock",
        "type": "bytes"
      },
      {
        "internalType": "uint256",
        "name": "epoch",
        "type": "uint256"
      }
    ],
    "name": "initialize",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "bytes",
        "name": "rlpHeader",
        "type": "bytes"
      }
    ],
    "name": "save",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "newImplementation",
        "type": "address"
      }
    ],
    "name": "upgradeTo",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [
      {
        "internalType": "address",
        "name": "newImplementation",
        "type": "address"
      },
      {
        "internalType": "bytes",
        "name": "data",
        "type": "bytes"
      }
    ],
    "name": "upgradeToAndCall",
    "outputs": [],
    "stateMutability": "payable",
    "type": "function"
  }
]`

const LightNode = `[
	{
		"inputs": [
			{
				"internalType": "uint256",
				"name": "_threshold",
				"type": "uint256"
			},
			{
				"internalType": "address[]",
				"name": "validaters",
				"type": "address[]"
			},
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "x",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "y",
						"type": "uint256"
					}
				],
				"internalType": "struct IBLSPoint.G1[]",
				"name": "_pairKeys",
				"type": "tuple[]"
			},
			{
				"internalType": "uint256[]",
				"name": "_weights",
				"type": "uint256[]"
			},
			{
				"internalType": "uint256",
				"name": "epoch",
				"type": "uint256"
			},
			{
				"internalType": "uint256",
				"name": "epochSize",
				"type": "uint256"
			}
		],
		"name": "initialize",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"components": [
					{
						"internalType": "bytes",
						"name": "parentHash",
						"type": "bytes"
					},
					{
						"internalType": "address",
						"name": "coinbase",
						"type": "address"
					},
					{
						"internalType": "bytes",
						"name": "root",
						"type": "bytes"
					},
					{
						"internalType": "bytes",
						"name": "txHash",
						"type": "bytes"
					},
					{
						"internalType": "bytes",
						"name": "receiptHash",
						"type": "bytes"
					},
					{
						"internalType": "bytes",
						"name": "bloom",
						"type": "bytes"
					},
					{
						"internalType": "uint256",
						"name": "number",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "gasLimit",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "gasUsed",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "time",
						"type": "uint256"
					},
					{
						"internalType": "bytes",
						"name": "extraData",
						"type": "bytes"
					},
					{
						"internalType": "bytes",
						"name": "mixDigest",
						"type": "bytes"
					},
					{
						"internalType": "bytes",
						"name": "nonce",
						"type": "bytes"
					},
					{
						"internalType": "uint256",
						"name": "baseFee",
						"type": "uint256"
					}
				],
				"internalType": "struct ILightNode.blockHeader",
				"name": "bh",
				"type": "tuple"
			},
			{
				"components": [
					{
						"internalType": "uint256",
						"name": "xr",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "xi",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "yr",
						"type": "uint256"
					},
					{
						"internalType": "uint256",
						"name": "yi",
						"type": "uint256"
					}
				],
				"internalType": "struct IBLSPoint.G2",
				"name": "aggPk",
				"type": "tuple"
			}
		],
		"name": "updateBlockHeader",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "bytes",
				"name": "_receiptProofBytes",
				"type": "bytes"
			}
		],
		"name": "verifyProofData",
		"outputs": [
			{
				"internalType": "bool",
				"name": "success",
				"type": "bool"
			},
			{
				"internalType": "string",
				"name": "message",
				"type": "string"
			},
			{
				"internalType": "bytes",
				"name": "logsHash",
				"type": "bytes"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "headerHeight",
		"outputs": [
		  {
			"internalType": "uint256",
			"name": "",
			"type": "uint256"
		  }
		],
		"stateMutability": "view",
		"type": "function"
  	}
]`

var (
	EncodeReceiptABI = `[
{
        "inputs":[
            {
                "components":[
                    {
                        "components":[
                            {
                                "internalType":"bytes",
                                "name":"parentHash",
                                "type":"bytes"
                            },
                            {
                                "internalType":"address",
                                "name":"coinbase",
                                "type":"address"
                            },
                            {
                                "internalType":"bytes",
                                "name":"root",
                                "type":"bytes"
                            },
                            {
                                "internalType":"bytes",
                                "name":"txHash",
                                "type":"bytes"
                            },
                            {
                                "internalType":"bytes",
                                "name":"receiptHash",
                                "type":"bytes"
                            },
                            {
                                "internalType":"bytes",
                                "name":"bloom",
                                "type":"bytes"
                            },
                            {
                                "internalType":"uint256",
                                "name":"number",
                                "type":"uint256"
                            },
                            {
                                "internalType":"uint256",
                                "name":"gasLimit",
                                "type":"uint256"
                            },
                            {
                                "internalType":"uint256",
                                "name":"gasUsed",
                                "type":"uint256"
                            },
                            {
                                "internalType":"uint256",
                                "name":"time",
                                "type":"uint256"
                            },
                            {
                                "internalType":"bytes",
                                "name":"extraData",
                                "type":"bytes"
                            },
                            {
                                "internalType":"bytes",
                                "name":"mixDigest",
                                "type":"bytes"
                            },
                            {
                                "internalType":"bytes",
                                "name":"nonce",
                                "type":"bytes"
                            },
                            {
                                "internalType":"uint256",
                                "name":"baseFee",
                                "type":"uint256"
                            }
                        ],
                        "internalType":"struct ILightNodePoint.blockHeader",
                        "name":"header",
                        "type":"tuple"
                    },
                    {
                        "components":[
                            {
                                "internalType":"uint256",
                                "name":"xr",
                                "type":"uint256"
                            },
                            {
                                "internalType":"uint256",
                                "name":"xi",
                                "type":"uint256"
                            },
                            {
                                "internalType":"uint256",
                                "name":"yr",
                                "type":"uint256"
                            },
                            {
                                "internalType":"uint256",
                                "name":"yi",
                                "type":"uint256"
                            }
                        ],
                        "internalType":"struct IBLSPoint.G2",
                        "name":"aggPk",
                        "type":"tuple"
                    },
                    {
                        "components":[
                            {
                                "internalType":"uint256",
                                "name":"receiptType",
                                "type":"uint256"
                            },
                            {
                                "internalType":"bytes",
                                "name":"postStateOrStatus",
                                "type":"bytes"
                            },
                            {
                                "internalType":"uint256",
                                "name":"cumulativeGasUsed",
                                "type":"uint256"
                            },
                            {
                                "internalType":"bytes",
                                "name":"bloom",
                                "type":"bytes"
                            },
                            {
                                "components":[
                                    {
                                        "internalType":"address",
                                        "name":"addr",
                                        "type":"address"
                                    },
                                    {
                                        "internalType":"bytes[]",
                                        "name":"topics",
                                        "type":"bytes[]"
                                    },
                                    {
                                        "internalType":"bytes",
                                        "name":"data",
                                        "type":"bytes"
                                    }
                                ],
                                "internalType":"struct ILightNodePoint.txLog[]",
                                "name":"logs",
                                "type":"tuple[]"
                            }
                        ],
                        "internalType":"struct ILightNodePoint.txReceipt",
                        "name":"receipt",
                        "type":"tuple"
                    },
                    {
                        "internalType":"bytes",
                        "name":"keyIndex",
                        "type":"bytes"
                    },
                    {
                        "internalType":"bytes[]",
                        "name":"proof",
                        "type":"bytes[]"
                    }
                ],
                "internalType":"struct ILightNodePoint.receiptProof",
                "name":"_receiptProof",
                "type":"tuple"
            }
        ],
        "name":"getBytes",
        "outputs":[
            {
                "internalType":"bytes",
                "name":"",
                "type":"bytes"
            }
        ],
        "stateMutability":"view",
        "type":"function"
    }
]`

	VerifyAbi = `[
		{
		  "inputs": [
			 {
				"internalType": "bytes",
				"name": "receiptProof",
				"type": "bytes"
			 }
		  ],
		  "name": "verifyProofData",
		  "outputs": [
			 {
				"internalType": "bool",
				"name": "success",
				"type": "bool"
			 },
			 {
				"internalType": "string",
				"name": "message",
				"type": "string"
			 },
			 {
				"internalType": "bytes",
				"name": "logs",
				"type": "bytes"
			 }
		  ],
		  "stateMutability": "nonpayable",
		  "type": "function"
	   }
	]`
)
