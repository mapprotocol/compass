async function main() {
    const [deployer] = await ethers.getSigners();
    console.log(
        "Deploying contracts with the account:",
        await deployer.getAddress()
    );
    console.log("Account balance:", (await deployer.getBalance()).toString());

    const MaticData = await ethers.getContractFactory("MaticData");
    const mData = await MaticData.deploy();
    await mData.deployed();
    console.log("MaticData address:", mData.address);

    const MaticStaking = await ethers.getContractFactory("MaticStaking");
    const mStaking = await MaticStaking.deploy(mData.address);
    await mStaking.deployed();
    console.log("MaticStaking address:", mStaking.address);

    await mData.addManager(mStaking.address);
    await mStaking.addManager("0x228F78fC398DB973B96eD666C92E78753b9466Eb")
}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });