/**
 * @type import('hardhat/config').HardhatUserConfig
 */

require("@nomiclabs/hardhat-waffle");

const PRIVATE_KEY = "6cd5c9865cc681c6d2c18856e7465b4261d0220548ae9bf00f26944e2362f2d3";

module.exports = {
  solidity: "0.8.0",
  networks: {
    HecoTest: {
      url: `https://http-testnet.hecochain.com`,
      chainId : 256,
      accounts: [`0x${PRIVATE_KEY}`]
    },
    MaticTest: {
      url: `https://rpc-mumbai.maticvigil.com/`,
      chainId : 80001,
      accounts: [`0x${PRIVATE_KEY}`]
    },
    Matic: {
      url: `https://rpc-mainnet.maticvigil.com`,
      chainId : 137,
      accounts: [`0x${PRIVATE_KEY}`]
    },
    Heco: {
      url: `https://http-mainnet-node.huobichain.com`,
      chainId : 128,
      accounts: [`0x${PRIVATE_KEY}`]
    }
  }
};
