// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title AkesaAuditLedger
 * @notice Permissioned audit ledger for Akesa Patient Data Exchange System.
 * @dev Conforms to SRS & SDD (FR-BC-01 to FR-BC-04):
 *      - Stores SHA-256 / cryptographic hashes of patient profiles (off-chain privacy / PDP compliant).
 *      - Records consent decisions (Pending, Approved, Rejected, Revoked).
 *      - Records data access events by hospital staff.
 *      - Verifies profile integrity before data disclosure.
 */
contract AkesaAuditLedger {
    address public owner;

    enum ConsentStatus {
        NONE,
        PENDING,
        APPROVED,
        REJECTED,
        REVOKED
    }

    struct ProfileRecord {
        bytes32 profileHash;
        uint256 version;
        uint256 updatedAt;
    }

    struct ConsentRecord {
        bytes32 requestId;
        bytes32 patientId;
        bytes32 hospitalId;
        ConsentStatus status;
        bytes32 metadataHash;
        uint256 updatedAt;
    }

    struct AccessLog {
        bytes32 requestId;
        bytes32 actorId;
        bytes32 dataHash;
        uint256 accessedAt;
    }

    // patientId => ProfileRecord
    mapping(bytes32 => ProfileRecord) public patientProfiles;

    // requestId => ConsentRecord
    mapping(bytes32 => ConsentRecord) public consents;

    // List of access logs for audit trail verification
    AccessLog[] public accessLogs;

    // Events emitted for public / off-chain monitoring & block explorer verification
    event ProfileHashRecorded(
        bytes32 indexed patientId,
        bytes32 profileHash,
        uint256 version,
        uint256 timestamp
    );

    event ConsentRecorded(
        bytes32 indexed requestId,
        bytes32 indexed patientId,
        bytes32 indexed hospitalId,
        uint8 status,
        bytes32 metadataHash,
        uint256 timestamp
    );

    event DataAccessRecorded(
        bytes32 indexed requestId,
        bytes32 indexed actorId,
        bytes32 dataHash,
        uint256 timestamp
    );

    event OwnershipTransferred(
        address indexed previousOwner,
        address indexed newOwner
    );

    modifier onlyOwner() {
        require(msg.sender == owner, "AkesaAuditLedger: caller is not the owner");
        _;
    }

    constructor() {
        owner = msg.sender;
        emit OwnershipTransferred(address(0), msg.sender);
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "AkesaAuditLedger: invalid new owner");
        emit OwnershipTransferred(owner, newOwner);
        owner = newOwner;
    }

    /**
     * @notice FR-BC-01: Record the cryptographic hash of a patient profile.
     * @param patientId UUID of the patient converted to bytes32.
     * @param profileHash Keyed SHA-256 / HMAC hash of canonical profile data.
     */
    function recordProfileHash(
        bytes32 patientId,
        bytes32 profileHash
    ) external onlyOwner {
        require(patientId != bytes32(0), "invalid patient ID");
        require(profileHash != bytes32(0), "invalid profile hash");

        ProfileRecord storage record = patientProfiles[patientId];
        record.profileHash = profileHash;
        record.version += 1;
        record.updatedAt = block.timestamp;

        emit ProfileHashRecorded(
            patientId,
            profileHash,
            record.version,
            block.timestamp
        );
    }

    /**
     * @notice FR-BC-02: Record consent decision (Pending, Approved, Rejected, Revoked).
     */
    function recordConsentDecision(
        bytes32 requestId,
        bytes32 patientId,
        bytes32 hospitalId,
        uint8 status,
        bytes32 metadataHash
    ) external onlyOwner {
        require(requestId != bytes32(0), "invalid request ID");
        require(status <= uint8(ConsentStatus.REVOKED), "invalid consent status");

        consents[requestId] = ConsentRecord({
            requestId: requestId,
            patientId: patientId,
            hospitalId: hospitalId,
            status: ConsentStatus(status),
            metadataHash: metadataHash,
            updatedAt: block.timestamp
        });

        emit ConsentRecorded(
            requestId,
            patientId,
            hospitalId,
            status,
            metadataHash,
            block.timestamp
        );
    }

    /**
     * @notice Record data access event when hospital staff reads patient data.
     */
    function recordDataAccess(
        bytes32 requestId,
        bytes32 actorId,
        bytes32 dataHash
    ) external onlyOwner {
        require(requestId != bytes32(0), "invalid request ID");

        accessLogs.push(AccessLog({
            requestId: requestId,
            actorId: actorId,
            dataHash: dataHash,
            accessedAt: block.timestamp
        }));

        emit DataAccessRecorded(
            requestId,
            actorId,
            dataHash,
            block.timestamp
        );
    }

    /**
     * @notice FR-BC-03: Verify if current real-time hash matches on-chain record.
     * @param patientId UUID of the patient converted to bytes32.
     * @param currentHash Real-time hash computed from current DB state.
     * @return isValid True if match, false if data was tampered off-chain.
     * @return recordedHash The hash stored on-chain.
     */
    function verifyProfileHash(
        bytes32 patientId,
        bytes32 currentHash
    ) external view returns (bool isValid, bytes32 recordedHash) {
        recordedHash = patientProfiles[patientId].profileHash;
        isValid = (recordedHash != bytes32(0) && recordedHash == currentHash);
    }

    /**
     * @notice Get latest profile record info.
     */
    function getProfile(
        bytes32 patientId
    ) external view returns (bytes32 profileHash, uint256 version, uint256 updatedAt) {
        ProfileRecord memory p = patientProfiles[patientId];
        return (p.profileHash, p.version, p.updatedAt);
    }

    /**
     * @notice Get total access logs count.
     */
    function getAccessLogsCount() external view returns (uint256) {
        return accessLogs.length;
    }
}
