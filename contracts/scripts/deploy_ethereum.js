async function main() {

    const [deployer] = await ethers.getSigners();

    console.log(
        "Deploying contracts with the account:",
        await deployer.getAddress()
    );

    console.log("Account balance:", (await deployer.getBalance()).toString());

    const EthData = await ethers.getContractFactory("EthereumData");
    const eData = await EthData.deploy();
    await eData.deployed();
    console.log("EthData address:", eData.address);

    const EthStaking = await ethers.getContractFactory("EthereumStaking");
    //test
    //const eStaking = await EthStaking.deploy(eData.address,"0x07cc6bbe1ea85a39ee3fe359750a553a906fbf4e");
    //mainnet
    const eStaking = await EthStaking.deploy(eData.address,"0x9E976F211daea0D652912AB99b0Dc21a7fD728e4");
    await eStaking.deployed();
    console.log("EthStaking address:", eStaking.address);

    await eData.addManager(eStaking.address);
    // await eStaking.addManager("0x228F78fC398DB973B96eD666C92E78753b9466Eb")
    await eStaking.addManager("0x200aee9ba7040d778922a763ce8a50948d61aff5");

}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });