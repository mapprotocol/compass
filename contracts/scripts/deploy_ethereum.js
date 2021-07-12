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
    const eStaking = await EthStaking.deploy(eData.address,"0x07cc6bbe1ea85a39ee3fe359750a553a906fbf4e");
    await eStaking.deployed();
    console.log("EthStaking address:", eStaking.address);

    await eData.addManager(eStaking.address);
    await eStaking.addManager("0x228F78fC398DB973B96eD666C92E78753b9466Eb")

}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });