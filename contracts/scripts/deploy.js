const hre = require("hardhat");

async function main() {
  console.log("Deploying AkesaAuditLedger...");
  const [deployer] = await hre.ethers.getSigners();
  console.log("Deployer address:", deployer.address);

  const AkesaAuditLedger = await hre.ethers.getContractFactory("AkesaAuditLedger");
  const ledger = await AkesaAuditLedger.deploy();
  await ledger.waitForDeployment();

  const contractAddress = await ledger.getAddress();
  console.log("AkesaAuditLedger deployed successfully to:", contractAddress);
  console.log("\nSet this address in your backend/.env:");
  console.log(`BLOCKCHAIN_CONTRACT_ADDRESS=${contractAddress}`);
  console.log(`BLOCKCHAIN_RPC_URL=http://127.0.0.1:8545`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
