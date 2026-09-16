const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("AkesaAuditLedger Smart Contract", function () {
  let ledger;
  let owner;
  let otherAccount;

  beforeEach(async function () {
    [owner, otherAccount] = await ethers.getSigners();
    const AkesaAuditLedger = await ethers.getContractFactory("AkesaAuditLedger");
    ledger = await AkesaAuditLedger.deploy();
    await ledger.waitForDeployment();
  });

  it("Should set the deployer as owner", async function () {
    expect(await ledger.owner()).to.equal(owner.address);
  });

  it("FR-BC-01: Should record patient profile hash and emit event", async function () {
    const patientId = ethers.keccak256(ethers.toUtf8Bytes("patient-123"));
    const profileHash = ethers.keccak256(ethers.toUtf8Bytes("canonical-profile-data-v1"));

    await expect(ledger.recordProfileHash(patientId, profileHash))
      .to.emit(ledger, "ProfileHashRecorded")
      .withArgs(patientId, profileHash, 1, (ts) => ts > 0);

    const [recordedHash, version, updatedAt] = await ledger.getProfile(patientId);
    expect(recordedHash).to.equal(profileHash);
    expect(version).to.equal(1);
    expect(updatedAt).to.be.gt(0);
  });

  it("FR-BC-02: Should record consent decision (Approved, Revoked)", async function () {
    const requestId = ethers.keccak256(ethers.toUtf8Bytes("request-abc"));
    const patientId = ethers.keccak256(ethers.toUtf8Bytes("patient-123"));
    const hospitalId = ethers.keccak256(ethers.toUtf8Bytes("hospital-xyz"));
    const metadataHash = ethers.keccak256(ethers.toUtf8Bytes("categories:identitas,rekam_medis"));

    // 2 = APPROVED
    await expect(ledger.recordConsentDecision(requestId, patientId, hospitalId, 2, metadataHash))
      .to.emit(ledger, "ConsentRecorded")
      .withArgs(requestId, patientId, hospitalId, 2, metadataHash, (ts) => ts > 0);

    const consent = await ledger.consents(requestId);
    expect(consent.status).to.equal(2);
    expect(consent.patientId).to.equal(patientId);
  });

  it("FR-BC-03 & FR-BC-04: Should verify valid hash and detect tampered data", async function () {
    const patientId = ethers.keccak256(ethers.toUtf8Bytes("patient-456"));
    const originalHash = ethers.keccak256(ethers.toUtf8Bytes("original-real-data"));
    const tamperedHash = ethers.keccak256(ethers.toUtf8Bytes("hacked-or-altered-data"));

    await ledger.recordProfileHash(patientId, originalHash);

    // FR-BC-03: Matching hash returns valid = true
    const [isValidOriginal, recorded1] = await ledger.verifyProfileHash(patientId, originalHash);
    expect(isValidOriginal).to.be.true;
    expect(recorded1).to.equal(originalHash);

    // FR-BC-04: Tampered hash returns valid = false
    const [isValidTampered, recorded2] = await ledger.verifyProfileHash(patientId, tamperedHash);
    expect(isValidTampered).to.be.false;
    expect(recorded2).to.equal(originalHash);
  });

  it("Should record data access event", async function () {
    const requestId = ethers.keccak256(ethers.toUtf8Bytes("request-abc"));
    const staffId = ethers.keccak256(ethers.toUtf8Bytes("staff-999"));
    const dataHash = ethers.keccak256(ethers.toUtf8Bytes("accessed-data-snapshot"));

    await expect(ledger.recordDataAccess(requestId, staffId, dataHash))
      .to.emit(ledger, "DataAccessRecorded")
      .withArgs(requestId, staffId, dataHash, (ts) => ts > 0);

    expect(await ledger.getAccessLogsCount()).to.equal(1);
  });

  it("Should prevent non-owner from recording transactions", async function () {
    const patientId = ethers.keccak256(ethers.toUtf8Bytes("patient-123"));
    const profileHash = ethers.keccak256(ethers.toUtf8Bytes("data"));

    await expect(
      ledger.connect(otherAccount).recordProfileHash(patientId, profileHash)
    ).to.be.revertedWith("AkesaAuditLedger: caller is not the owner");
  });
});
