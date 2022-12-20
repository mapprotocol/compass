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

const (
	NearAbiJson = `
	[
		{
			"inputs": [
			  {
				"internalType": "bytes",
				"name": "head",
				"type": "bytes"
			  },
			  {
				"internalType": "bytes",
				"name": "proof",
				"type": "bytes"
			  }
			],
			"name": "getBytes",
			"outputs": [
			  {
				"internalType": "bytes",
				"name": "_receiptProof",
				"type": "bytes"
			  }
			],
			"stateMutability": "view",
			"type": "function"
		},
		{
		  "inputs": [
			{
			  "internalType": "bytes",
			  "name": "_receiptProof",
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
		  "stateMutability": "view",
		  "type": "function"
		}
	]`
	McsAbi = `[
		{
		  "inputs": [
			{
			  "internalType": "bytes32",
			  "name": "",
			  "type": "bytes32"
			}
		  ],
		  "name": "orderList",
		  "outputs": [
			{
			  "internalType": "bool",
			  "name": "",
			  "type": "bool"
			}
		  ],
		  "stateMutability": "view",
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
			  "internalType": "bytes",
			  "name": "_receiptProof",
			  "type": "bytes"
			}
		  ],
		  "name": "swapIn",
		  "outputs": [],
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
			"inputs": [
				{
					"internalType": "uint256",
					"name": "_fromChain",
					"type": "uint256"
				},
				{
					"internalType": "bytes",
					"name": "receiptProof",
					"type": "bytes"
				}
			],
			"name": "depositIn",
			"outputs": [],
			"stateMutability": "payable",
			"type": "function"
		},
		{
			"inputs":[
				{
					"internalType":"uint256",
					"name":"",
					"type":"uint256"
				},
				{
					"internalType":"bytes",
					"name":"receiptProof",
					"type":"bytes"
				}
			],
			"name":"transferIn",
			"outputs":[
		
			],
			"stateMutability":"nonpayable",
			"type":"function"
		},{
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
									"internalType":"address[]",
									"name":"validators",
									"type":"address[]"
								},
								{
									"internalType":"bytes[]",
									"name":"addedPubKey",
									"type":"bytes[]"
								},
								{
									"internalType":"bytes[]",
									"name":"addedG1PubKey",
									"type":"bytes[]"
								},
								{
									"internalType":"uint256",
									"name":"removeList",
									"type":"uint256"
								},
								{
									"internalType":"bytes",
									"name":"seal",
									"type":"bytes"
								},
								{
									"components":[
										{
											"internalType":"uint256",
											"name":"bitmap",
											"type":"uint256"
										},
										{
											"internalType":"bytes",
											"name":"signature",
											"type":"bytes"
										},
										{
											"internalType":"uint256",
											"name":"round",
											"type":"uint256"
										}
									],
									"internalType":"struct ILightNodePoint.istanbulAggregatedSeal",
									"name":"aggregatedSeal",
									"type":"tuple"
								},
								{
									"components":[
										{
											"internalType":"uint256",
											"name":"bitmap",
											"type":"uint256"
										},
										{
											"internalType":"bytes",
											"name":"signature",
											"type":"bytes"
										},
										{
											"internalType":"uint256",
											"name":"round",
											"type":"uint256"
										}
									],
									"internalType":"struct ILightNodePoint.istanbulAggregatedSeal",
									"name":"parentAggregatedSeal",
									"type":"tuple"
								}
							],
							"internalType":"struct ILightNodePoint.istanbulExtra",
							"name":"ist",
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
									"name":"receiptRlp",
									"type":"bytes"
								}
							],
							"internalType":"struct ILightNodePoint.TxReceiptRlp",
							"name":"txReceiptRlp",
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
			"stateMutability":"pure",
			"type":"function"
		}
	]`
	LightMangerAbi = `[
		{
			"inputs":[
				{
					"internalType":"uint256",
					"name":"_chainId",
					"type":"uint256"
				},
				{
					"internalType":"bytes",
					"name":"_blockHeader",
					"type":"bytes"
				}
			],
			"name":"updateBlockHeader",
			"outputs":[
		
			],
			"stateMutability":"nonpayable",
			"type":"function"
		},
		{
			"inputs": [
				{
					"internalType": "uint256",
					"name": "_chainId",
					"type": "uint256"
				}
			],
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
		},
		{
			"inputs":[
				{
					"internalType":"uint256",
					"name":"_chainId",
					"type":"uint256"
				},
				{
					"internalType":"bytes",
					"name":"_receiptProof",
					"type":"bytes"
				}
			],
			"name":"verifyProofData",
			"outputs":[
				{
					"internalType":"bool",
					"name":"success",
					"type":"bool"
				},
				{
					"internalType":"string",
					"name":"message",
					"type":"string"
				},
				{
					"internalType":"bytes",
					"name":"logs",
					"type":"bytes"
				}
			],
			"stateMutability":"view",
			"type":"function"
		},
		{
		  "inputs": [
			{
			  "internalType": "uint256",
			  "name": "_chainId",
			  "type": "uint256"
			}
		  ],
		  "name": "verifiableHeaderRange",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "left",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "right",
			  "type": "uint256"
			}
		  ],
		  "stateMutability": "view",
		  "type": "function"
		}
	]`
	BscAbiJson = `
	[
		{
			"inputs": [
				{
					"internalType": "bytes",
					"name": "_data",
					"type": "bytes"
				}
			],
			"name": "updateLightClient",
			"outputs": [],
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"inputs": [
				{
				  "internalType": "bytes",
				  "name": "_blockHeadersBytes",
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
			  "components": [
				{
				  "internalType": "bytes",
				  "name": "parentHash",
				  "type": "bytes"
				},
				{
				  "internalType": "bytes",
				  "name": "sha3Uncles",
				  "type": "bytes"
				},
				{
				  "internalType": "address",
				  "name": "miner",
				  "type": "address"
				},
				{
				  "internalType": "bytes",
				  "name": "stateRoot",
				  "type": "bytes"
				},
				{
				  "internalType": "bytes",
				  "name": "transactionsRoot",
				  "type": "bytes"
				},
				{
				  "internalType": "bytes",
				  "name": "receiptsRoot",
				  "type": "bytes"
				},
				{
				  "internalType": "bytes",
				  "name": "logsBloom",
				  "type": "bytes"
				},
				{
				  "internalType": "uint256",
				  "name": "difficulty",
				  "type": "uint256"
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
				  "name": "timestamp",
				  "type": "uint256"
				},
				{
				  "internalType": "bytes",
				  "name": "extraData",
				  "type": "bytes"
				},
				{
				  "internalType": "bytes",
				  "name": "mixHash",
				  "type": "bytes"
				},
				{
				  "internalType": "bytes",
				  "name": "nonce",
				  "type": "bytes"
				}
			  ],
			  "internalType": "struct Verify.BlockHeader[]",
			  "name": "_blockHeaders",
			  "type": "tuple[]"
			}
			],
			"name": "getHeadersBytes",
			"outputs": [
			{
			  "internalType": "bytes",
			  "name": "",
			  "type": "bytes"
			}
			],
			"stateMutability": "pure",
			"type": "function"
		},
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
									"internalType":"bytes",
									"name":"sha3Uncles",
									"type":"bytes"
								},
								{
									"internalType":"address",
									"name":"miner",
									"type":"address"
								},
								{
									"internalType":"bytes",
									"name":"stateRoot",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"transactionsRoot",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"receiptsRoot",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"logsBloom",
									"type":"bytes"
								},
								{
									"internalType":"uint256",
									"name":"difficulty",
									"type":"uint256"
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
									"name":"timestamp",
									"type":"uint256"
								},
								{
									"internalType":"bytes",
									"name":"extraData",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"mixHash",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"nonce",
									"type":"bytes"
								}
							],
							"internalType":"struct Verify.BlockHeader[]",
							"name":"headers",
							"type":"tuple[]"
						},
						{
							"components":[
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
											"internalType":"struct Verify.TxLog[]",
											"name":"logs",
											"type":"tuple[]"
										}
									],
									"internalType":"struct Verify.TxReceipt",
									"name":"txReceipt",
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
							"internalType":"struct Verify.ReceiptProof",
							"name":"receiptProof",
							"type":"tuple"
						}
					],
					"internalType":"struct LightNode.ProofData",
					"name":"proof",
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
			"stateMutability":"pure",
			"type":"function"
		}
	]
	`
	KlaytnAbiJson = `[
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
			  "name": "reward",
			  "type": "address"
			},
			{
			  "internalType": "bytes",
			  "name": "stateRoot",
			  "type": "bytes"
			},
			{
			  "internalType": "bytes",
			  "name": "transactionsRoot",
			  "type": "bytes"
			},
			{
			  "internalType": "bytes",
			  "name": "receiptsRoot",
			  "type": "bytes"
			},
			{
			  "internalType": "bytes",
			  "name": "logsBloom",
			  "type": "bytes"
			},
			{
			  "internalType": "uint256",
			  "name": "blockScore",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "number",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "gasUsed",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "timestamp",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "timestampFoS",
			  "type": "uint256"
			},
			{
			  "internalType": "bytes",
			  "name": "extraData",
			  "type": "bytes"
			},
			{
			  "internalType": "bytes",
			  "name": "governanceData",
			  "type": "bytes"
			},
			{
			  "internalType": "bytes",
			  "name": "voteData",
			  "type": "bytes"
			},
			{
			  "internalType": "uint256",
			  "name": "baseFee",
			  "type": "uint256"
			}
		  ],
		  "internalType": "struct ILightNodePoint.BlockHeader[]",
		  "name": "_blockHeaders",
		  "type": "tuple[]"
		}
	  ],
	  "name": "getHeadersBytes",
	  "outputs": [
		{
		  "internalType": "bytes",
		  "name": "",
		  "type": "bytes"
		}
	  ],
	  "stateMutability": "pure",
	  "type": "function"
	},
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
								"name":"reward",
								"type":"address"
							},
							{
								"internalType":"bytes",
								"name":"stateRoot",
								"type":"bytes"
							},
							{
								"internalType":"bytes",
								"name":"transactionsRoot",
								"type":"bytes"
							},
							{
								"internalType":"bytes",
								"name":"receiptsRoot",
								"type":"bytes"
							},
							{
								"internalType":"bytes",
								"name":"logsBloom",
								"type":"bytes"
							},
							{
								"internalType":"uint256",
								"name":"blockScore",
								"type":"uint256"
							},
							{
								"internalType":"uint256",
								"name":"number",
								"type":"uint256"
							},
							{
								"internalType":"uint256",
								"name":"gasUsed",
								"type":"uint256"
							},
							{
								"internalType":"uint256",
								"name":"timestamp",
								"type":"uint256"
							},
							{
								"internalType":"uint256",
								"name":"timestampFoS",
								"type":"uint256"
							},
							{
								"internalType":"bytes",
								"name":"extraData",
								"type":"bytes"
							},
							{
								"internalType":"bytes",
								"name":"governanceData",
								"type":"bytes"
							},
							{
								"internalType":"bytes",
								"name":"voteData",
								"type":"bytes"
							},
							{
								"internalType":"uint256",
								"name":"baseFee",
								"type":"uint256"
							}
						],
						"internalType":"struct ILightNodePoint.BlockHeader",
						"name":"header",
						"type":"tuple"
					},
					{
						"internalType":"bytes[]",
						"name":"receipts",
						"type":"bytes[]"
					},
					{
						"internalType":"uint256",
						"name":"logIndex",
						"type":"uint256"
					}
				],
				"internalType":"struct ILightNodePoint.ReceiptProof",
				"name":"_proof",
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
		"stateMutability":"pure",
		"type":"function"
	}]
	`
	HeightAbiJson = `
	[
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
	]
	`
	VerifiableHeaderRangeAbiJson = `
	[
		{
		  "inputs": [],
		  "name": "verifiableHeaderRange",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "left",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "right",
			  "type": "uint256"
			}
		  ],
		  "stateMutability": "view",
		  "type": "function"
		}
	]
	`
	Map2OtherAbi = `[{
		"inputs":[
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
				"name":"bh",
				"type":"tuple"
			},
			{
				"components":[
					{
						"internalType":"address[]",
						"name":"validators",
						"type":"address[]"
					},
					{
						"internalType":"bytes[]",
						"name":"addedPubKey",
						"type":"bytes[]"
					},
					{
						"internalType":"bytes[]",
						"name":"addedG1PubKey",
						"type":"bytes[]"
					},
					{
						"internalType":"uint256",
						"name":"removeList",
						"type":"uint256"
					},
					{
						"internalType":"bytes",
						"name":"seal",
						"type":"bytes"
					},
					{
						"components":[
							{
								"internalType":"uint256",
								"name":"bitmap",
								"type":"uint256"
							},
							{
								"internalType":"bytes",
								"name":"signature",
								"type":"bytes"
							},
							{
								"internalType":"uint256",
								"name":"round",
								"type":"uint256"
							}
						],
						"internalType":"struct ILightNodePoint.istanbulAggregatedSeal",
						"name":"aggregatedSeal",
						"type":"tuple"
					},
					{
						"components":[
							{
								"internalType":"uint256",
								"name":"bitmap",
								"type":"uint256"
							},
							{
								"internalType":"bytes",
								"name":"signature",
								"type":"bytes"
							},
							{
								"internalType":"uint256",
								"name":"round",
								"type":"uint256"
							}
						],
						"internalType":"struct ILightNodePoint.istanbulAggregatedSeal",
						"name":"parentAggregatedSeal",
						"type":"tuple"
					}
				],
				"internalType":"struct ILightNodePoint.istanbulExtra",
				"name":"ist",
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
			}
		],
		"name":"updateBlockHeader",
		"outputs":[
	
		],
		"stateMutability":"nonpayable",
		"type":"function"
	}]`
	MaticAbiJson = `[
		{
		  "inputs": [],
		  "name": "confirms",
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
			"inputs":[
				{
					"components":[
						{
							"internalType":"bytes",
							"name":"parentHash",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"sha3Uncles",
							"type":"bytes"
						},
						{
							"internalType":"address",
							"name":"miner",
							"type":"address"
						},
						{
							"internalType":"bytes",
							"name":"stateRoot",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"transactionsRoot",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"receiptsRoot",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"logsBloom",
							"type":"bytes"
						},
						{
							"internalType":"uint256",
							"name":"difficulty",
							"type":"uint256"
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
							"name":"timestamp",
							"type":"uint256"
						},
						{
							"internalType":"bytes",
							"name":"extraData",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"mixHash",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"nonce",
							"type":"bytes"
						},
						{
							"internalType":"uint256",
							"name":"baseFeePerGas",
							"type":"uint256"
						}
					],
					"internalType":"struct Verify.BlockHeader[]",
					"name":"_blockHeaders",
					"type":"tuple[]"
				}
			],
			"name":"getHeadersBytes",
			"outputs":[
				{
					"internalType":"bytes",
					"name":"",
					"type":"bytes"
				}
			],
			"stateMutability":"pure",
			"type":"function"
		},{
			  "inputs": [],
			  "name": "maxCanVerifyNum",
			  "outputs": [
				{
				  "internalType": "uint256",
				  "name": "",
				  "type": "uint256"
				}
			  ],
			  "stateMutability": "view",
			  "type": "function"
			},{
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
										"internalType":"bytes",
										"name":"sha3Uncles",
										"type":"bytes"
									},
									{
										"internalType":"address",
										"name":"miner",
										"type":"address"
									},
									{
										"internalType":"bytes",
										"name":"stateRoot",
										"type":"bytes"
									},
									{
										"internalType":"bytes",
										"name":"transactionsRoot",
										"type":"bytes"
									},
									{
										"internalType":"bytes",
										"name":"receiptsRoot",
										"type":"bytes"
									},
									{
										"internalType":"bytes",
										"name":"logsBloom",
										"type":"bytes"
									},
									{
										"internalType":"uint256",
										"name":"difficulty",
										"type":"uint256"
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
										"name":"timestamp",
										"type":"uint256"
									},
									{
										"internalType":"bytes",
										"name":"extraData",
										"type":"bytes"
									},
									{
										"internalType":"bytes",
										"name":"mixHash",
										"type":"bytes"
									},
									{
										"internalType":"bytes",
										"name":"nonce",
										"type":"bytes"
									},
									{
										"internalType":"uint256",
										"name":"baseFeePerGas",
										"type":"uint256"
									}
								],
								"internalType":"struct Verify.BlockHeader[]",
								"name":"headers",
								"type":"tuple[]"
							},
							{
								"components":[
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
												"internalType":"struct Verify.TxLog[]",
												"name":"logs",
												"type":"tuple[]"
											}
										],
										"internalType":"struct Verify.TxReceipt",
										"name":"txReceipt",
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
								"internalType":"struct Verify.ReceiptProof",
								"name":"receiptProof",
								"type":"tuple"
							}
						],
						"internalType":"struct LightNode.ProofData",
						"name":"_proof",
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
				"stateMutability":"pure",
				"type":"function"
			}
	]`
	Eth2AbiJson = `[
		{
			"inputs":[
				{
					"components":[
						{
							"components":[
								{
									"internalType":"uint64",
									"name":"slot",
									"type":"uint64"
								},
								{
									"internalType":"uint64",
									"name":"proposerIndex",
									"type":"uint64"
								},
								{
									"internalType":"bytes32",
									"name":"parentRoot",
									"type":"bytes32"
								},
								{
									"internalType":"bytes32",
									"name":"stateRoot",
									"type":"bytes32"
								},
								{
									"internalType":"bytes32",
									"name":"bodyRoot",
									"type":"bytes32"
								}
							],
							"internalType":"struct Types.BeaconBlockHeader",
							"name":"attestedHeader",
							"type":"tuple"
						},
						{
							"components":[
								{
									"internalType":"bytes",
									"name":"pubkeys",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"aggregatePubkey",
									"type":"bytes"
								}
							],
							"internalType":"struct Types.SyncCommittee",
							"name":"nextSyncCommittee",
							"type":"tuple"
						},
						{
							"internalType":"bytes32[]",
							"name":"nextSyncCommitteeBranch",
							"type":"bytes32[]"
						},
						{
							"components":[
								{
									"internalType":"uint64",
									"name":"slot",
									"type":"uint64"
								},
								{
									"internalType":"uint64",
									"name":"proposerIndex",
									"type":"uint64"
								},
								{
									"internalType":"bytes32",
									"name":"parentRoot",
									"type":"bytes32"
								},
								{
									"internalType":"bytes32",
									"name":"stateRoot",
									"type":"bytes32"
								},
								{
									"internalType":"bytes32",
									"name":"bodyRoot",
									"type":"bytes32"
								}
							],
							"internalType":"struct Types.BeaconBlockHeader",
							"name":"finalizedHeader",
							"type":"tuple"
						},
						{
							"internalType":"bytes32[]",
							"name":"finalityBranch",
							"type":"bytes32[]"
						},
						{
							"components":[
								{
									"internalType":"bytes",
									"name":"parentHash",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"sha3Uncles",
									"type":"bytes"
								},
								{
									"internalType":"address",
									"name":"miner",
									"type":"address"
								},
								{
									"internalType":"bytes",
									"name":"stateRoot",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"transactionsRoot",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"receiptsRoot",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"logsBloom",
									"type":"bytes"
								},
								{
									"internalType":"uint256",
									"name":"difficulty",
									"type":"uint256"
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
									"name":"timestamp",
									"type":"uint256"
								},
								{
									"internalType":"bytes",
									"name":"extraData",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"mixHash",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"nonce",
									"type":"bytes"
								},
								{
									"internalType":"uint256",
									"name":"baseFeePerGas",
									"type":"uint256"
								}
							],
							"internalType":"struct Types.BlockHeader",
							"name":"finalizedExeHeader",
							"type":"tuple"
						},
						{
							"internalType":"bytes32[]",
							"name":"exeFinalityBranch",
							"type":"bytes32[]"
						},
						{
							"components":[
								{
									"internalType":"bytes",
									"name":"syncCommitteeBits",
									"type":"bytes"
								},
								{
									"internalType":"bytes",
									"name":"syncCommitteeSignature",
									"type":"bytes"
								}
							],
							"internalType":"struct Types.SyncAggregate",
							"name":"syncAggregate",
							"type":"tuple"
						},
						{
							"internalType":"uint64",
							"name":"signatureSlot",
							"type":"uint64"
						}
					],
					"internalType":"struct Types.LightClientUpdate",
					"name":"_update",
					"type":"tuple"
				}
			],
			"name":"getUpdateBytes",
			"outputs":[
				{
					"internalType":"bytes",
					"name":"",
					"type":"bytes"
				}
			],
			"stateMutability":"pure",
			"type":"function"
		},
		{
			"inputs":[
				{
					"components":[
						{
							"internalType":"bytes",
							"name":"parentHash",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"sha3Uncles",
							"type":"bytes"
						},
						{
							"internalType":"address",
							"name":"miner",
							"type":"address"
						},
						{
							"internalType":"bytes",
							"name":"stateRoot",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"transactionsRoot",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"receiptsRoot",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"logsBloom",
							"type":"bytes"
						},
						{
							"internalType":"uint256",
							"name":"difficulty",
							"type":"uint256"
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
							"name":"timestamp",
							"type":"uint256"
						},
						{
							"internalType":"bytes",
							"name":"extraData",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"mixHash",
							"type":"bytes"
						},
						{
							"internalType":"bytes",
							"name":"nonce",
							"type":"bytes"
						},
						{
							"internalType":"uint256",
							"name":"baseFeePerGas",
							"type":"uint256"
						}
					],
					"internalType":"struct Types.BlockHeader[]",
					"name":"_headers",
					"type":"tuple[]"
				}
			],
			"name":"getHeadersBytes",
			"outputs":[
				{
					"internalType":"bytes",
					"name":"",
					"type":"bytes"
				}
			],
			"stateMutability":"pure",
			"type":"function"
		}
	]`
)
