async function main() {
    const [deployer] = await ethers.getSigners();

    console.log(
        "Deploying contracts with the account:",
        await deployer.getAddress()
    );

    console.log("Account balance:", (await deployer.getBalance()).toString());

    // const EthData = await ethers.getContractFactory("EthereumData");
    // const eData = await EthData.deploy();
    // await eData.deployed();
    // console.log("EthData address:", eData.address);

    const EthStaking = await ethers.getContractFactory("EthereumStaking");
    const eStaking = await EthStaking.deploy("0x9c6190c02E30D0a8dB5F9F39C8B4d3AF513C5C16","0x9E976F211daea0D652912AB99b0Dc21a7fD728e4");
    await eStaking.deployed();
    console.log("EthStaking address:", eStaking.address);

    await eData.addManager(eStaking.address);
    await eStaking.addManager("0x200aee9ba7040d778922a763ce8a50948d61aff5");

}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });