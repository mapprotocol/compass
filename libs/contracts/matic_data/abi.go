package matic_data

var curAbi = `
[
 {
  "inputs": [],
  "stateMutability": "nonpayable",
  "type": "constructor"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "_address",
    "type": "address"
   }
  ],
  "name": "addManager",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "",
    "type": "address"
   }
  ],
  "name": "addressBind",
  "outputs": [
   {
    "internalType": "address",
    "name": "",
    "type": "address"
   }
  ],
  "stateMutability": "view",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "",
    "type": "address"
   }
  ],
  "name": "bindAddress",
  "outputs": [
   {
    "internalType": "address",
    "name": "",
    "type": "address"
   }
  ],
  "stateMutability": "view",
  "type": "function"
 },
 {
  "inputs": [],
  "name": "get24HourSign",
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
  "name": "getAddressCount",
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
    "name": "_source",
    "type": "address"
   }
  ],
  "name": "getBindAddress",
  "outputs": [
   {
    "internalType": "address",
    "name": "",
    "type": "address"
   }
  ],
  "stateMutability": "view",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "_sender",
    "type": "address"
   }
  ],
  "name": "getLastSign",
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
  "name": "getStakingAmount",
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
    "name": "_sender",
    "type": "address"
   }
  ],
  "name": "getUserInfo",
  "outputs": [
   {
    "internalType": "uint256",
    "name": "amount",
    "type": "uint256"
   },
   {
    "internalType": "uint256",
    "name": "dayCount",
    "type": "uint256"
   },
   {
    "internalType": "uint256",
    "name": "daySign",
    "type": "uint256"
   },
   {
    "internalType": "uint256",
    "name": "stakingStatus",
    "type": "uint256"
   },
   {
    "internalType": "uint256[]",
    "name": "signTm",
    "type": "uint256[]"
   }
  ],
  "stateMutability": "view",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "_sender",
    "type": "address"
   }
  ],
  "name": "getUserInfos",
  "outputs": [
   {
    "components": [
     {
      "internalType": "uint256",
      "name": "stakingStatus",
      "type": "uint256"
     },
     {
      "internalType": "uint256",
      "name": "dayCount",
      "type": "uint256"
     },
     {
      "internalType": "uint256",
      "name": "daySign",
      "type": "uint256"
     },
     {
      "internalType": "uint256",
      "name": "amount",
      "type": "uint256"
     },
     {
      "internalType": "uint256[]",
      "name": "signTm",
      "type": "uint256[]"
     }
    ],
    "internalType": "struct matic_data.userInfo",
    "name": "u",
    "type": "tuple"
   }
  ],
  "stateMutability": "view",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "_address",
    "type": "address"
   }
  ],
  "name": "removeManager",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "uint256",
    "name": "count",
    "type": "uint256"
   }
  ],
  "name": "setAddressCount",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "_source",
    "type": "address"
   },
   {
    "internalType": "address",
    "name": "_bind",
    "type": "address"
   }
  ],
  "name": "setBindAddress",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "uint256",
    "name": "amount",
    "type": "uint256"
   }
  ],
  "name": "setStakingAmount",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "uint256",
    "name": "_dayCount",
    "type": "uint256"
   },
   {
    "internalType": "uint256",
    "name": "_daySign",
    "type": "uint256"
   },
   {
    "internalType": "uint256",
    "name": "_amount",
    "type": "uint256"
   },
   {
    "internalType": "address",
    "name": "_sender",
    "type": "address"
   }
  ],
  "name": "setUserInfo",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "_sender",
    "type": "address"
   },
   {
    "internalType": "uint256",
    "name": "status",
    "type": "uint256"
   }
  ],
  "name": "setUserWithdraw",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
 },
 {
  "inputs": [
   {
    "internalType": "address",
    "name": "_sender",
    "type": "address"
   },
   {
    "internalType": "uint256",
    "name": "day",
    "type": "uint256"
   },
   {
    "internalType": "uint256",
    "name": "hour",
    "type": "uint256"
   }
  ],
  "name": "sign",
  "outputs": [
   {
    "internalType": "uint256",
    "name": "",
    "type": "uint256"
   }
  ],
  "stateMutability": "nonpayable",
  "type": "function"
 }
]
`
