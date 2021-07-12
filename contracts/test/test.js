const { expect } = require("chai");

describe("Token contract", function() {
    it("Deployment should assign the total supply of tokens to the owner", async function() {
        const [owner] = await ethers.getSigners();

        const EthData = await ethers.getContractFactory("EthData");

        const hardhatEthData = await EthData.deploy();
        await hardhatEthData.deployed();

        const info = await hardhatEthData.getUserInfo(owner.getAddress());
        console.log(hardhatEthData.address);
    });
});