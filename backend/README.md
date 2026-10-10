# Akesa Backend

Backend untuk **Akesa**, aplikasi pendaftaran pasien antar-rumah sakit yang memungkinkan pasien menyimpan data secara terpusat dan memberikan atau mencabut izin akses data kepada rumah sakit.

Backend ini dibangun menggunakan:

* **Go 1.25**
* **PostgreSQL 17**
* **Clerk** untuk autentikasi
* **AES-256-GCM** untuk enkripsi field sensitif
* **HMAC-SHA256** untuk hash yang digunakan pada audit trail
* **Hash chain** sebagai lapisan audit trail
* **EVM/Solidity** sebagai bagian dari hybrid blockchain architecture
* **Docker** untuk environment PostgreSQL lokal

Fokus utama backend:

1. Authentication & authorization
2. Patient profile
3. Identity / KTP verification
4. Hospital management
5. Access request & consent
6. Patient QR credential
7. Audit trail & integrity verification
8. Hybrid blockchain proof of concept
9. Encryption untuk data sensitif

---

# 1. Cara Menjalankan di Lokal

## 1.1. Environment

Buat file `.env` dari template:

```bash
cp .env.example .env
```

Kemudian isi konfigurasi yang diperlukan, terutama:

```env
CLERK_SECRET_KEY=...

DATABASE_URL=postgres://akesa:akesa@localhost:5432/akesa?sslmode=disable
PORT=8080
```

Untuk fitur enkripsi dan audit hash, generate key berikut:

```bash
openssl rand -base64 32
```

Gunakan hasilnya sebagai:

```env
FIELD_ENCRYPTION_KEY=...
```

Kemudian:

```bash
openssl rand -hex 32
```

Gunakan hasil pertama sebagai:

```env
NIK_HASH_KEY=...
```

Generate satu key lagi:

```bash
openssl rand -hex 32
```

Gunakan hasil kedua sebagai:

```env
AUDIT_HASH_KEY=...
```

**Jangan menggunakan key yang sama untuk ketiga konfigurasi tersebut.**

---

## 1.2. Jalankan PostgreSQL

Jika menggunakan Docker:

```bash
docker compose up -d
```

Cek:

```bash
docker compose ps
```

Pastikan container PostgreSQL sudah `Up`.

---

## 1.3. Jalankan Migration

Pastikan `golang-migrate` sudah terinstall.

Jika belum:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Kemudian:

```bash
make migrate-up
```

Atau jalankan migration sesuai konfigurasi lokal.

Migration saat ini mencakup:

```text
000001_create_users
000002_create_hospitals
000003_create_patient_profiles
000004_create_identity_verifications
000005_create_hospital_staff
000006_create_access_requests
000007_create_audit_logs
000008_encrypt_sensitive_fields
000009_create_patient_qr_credentials
000010_add_hospital_registration_fields
000011_create_hospital_invitations
000012_create_hospital_applications
```

---

## 1.4. Jalankan API

```bash
make run
```

Default API:

```text
http://localhost:8080
```

Test:

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "ok"
}
```

Jika response tersebut muncul, API sudah berjalan.

---

# 2. Struktur Project

Struktur utama backend:

```text
backend/
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── access/
│   ├── audit/
│   ├── auth/
│   ├── blockchain/
│   ├── config/
│   ├── crypto/
│   ├── hospital/
│   ├── middleware/
│   ├── patient/
│   │   └── identify/
│   ├── patientqr/
│   ├── server/
│   └── user/
│
├── migrations/
│
├── storage/
│   └── identity/
│
├── .env.example
├── go.mod
└── go.sum
```

Setiap domain umumnya memiliki pola:

```text
model.go
dto.go
errors.go
repository.go
service.go
handler.go
```

Tidak semua package harus memiliki seluruh file tersebut.

Alur request:

```text
HTTP Request
     │
     ▼
   Handler
     │
     ▼
   Service
     │
     ▼
 Repository
     │
     ▼
 PostgreSQL
```

`cmd/api/main.go` digunakan untuk melakukan wiring antar-komponen.

Routing HTTP berada di:

```text
internal/server/server.go
```

---

# 3. Auth & Role

Akesa menggunakan **Clerk** untuk authentication.

Mobile app mengirim JWT Clerk melalui:

```http
Authorization: Bearer <clerk_jwt_token>
```

Backend kemudian melakukan:

```text
Clerk JWT
   │
   ▼
auth.RequireAuth
   │
   ▼
auth.WithUser
   │
   ▼
Load user dari PostgreSQL
   │
   ▼
AuthUser
   │
   ├── ID
   ├── Role
   └── Status
```

Role aplikasi disimpan di database sendiri:

```text
PATIENT
HOSPITAL_STAFF
ADMIN
```

Role **tidak boleh dipercaya dari request body atau data yang dikirim client**.

Untuk endpoint yang membutuhkan role tertentu, gunakan:

```go
s.chain(deps, handler, RoleAdmin)
```

atau:

```go
s.chain(deps, handler, RoleStaff)
```

Urutan middleware:

```text
auth.RequireAuth
        ↓
auth.WithUser
        ↓
auth.RequireRole(...)
```

Contoh:

```go
s.mux.Handle(
    "PATCH /api/v1/admin/hospitals/{id}/verify",
    s.chain(
        deps,
        http.HandlerFunc(deps.HospitalHandler.Verify),
        RoleAdmin,
    ),
)
```

Selain pengecekan role, endpoint yang berkaitan dengan resource milik user juga harus melakukan pengecekan ownership di service layer.

Contohnya:

```text
Patient A
   │
   └── Access Request A

Patient B tidak boleh approve
Access Request A
```

---

# 4. Audit Trail & Hybrid Blockchain

Akesa memiliki audit trail yang digunakan untuk mencatat aktivitas penting dalam sistem.

Audit trail menggunakan **hash chain**.

Secara sederhana:

```text
Block 1
  │
  ▼
Block 2
  │
  ▼
Block 3
  │
  ▼
Block 4
```

Setiap entry memiliki hubungan dengan hash entry sebelumnya.

Konsep sederhananya:

```text
currentHash = HMAC(previousHash + data)
```

Jika satu entry lama diubah, hash setelahnya akan ikut berubah sehingga integritas rantai dapat diverifikasi.

Data pribadi pasien **tidak dimasukkan langsung ke blockchain/audit chain**.

Yang dicatat terutama adalah:

* event/action
* actor
* timestamp
* resource
* hash
* metadata yang diperlukan untuk audit

---

## 4.1. Hybrid Blockchain

Backend juga memiliki package:

```text
internal/blockchain/
```

dan smart contract:

```text
contracts/
└── contracts/
    └── AkesaAuditLedger.sol
```

Arsitektur yang digunakan:

```text
                Akesa Backend
                     │
          ┌──────────┴──────────┐
          │                     │
          ▼                     ▼
   PostgreSQL Hash Chain     EVM Blockchain
          │                     │
          └──────────┬──────────┘
                     ▼
              Integrity Check
```

PostgreSQL digunakan untuk menyimpan audit trail aplikasi, sedangkan blockchain digunakan sebagai lapisan tambahan untuk membuktikan integritas data audit.

Implementasi blockchain ini masih merupakan bagian dari **proof of concept**, sehingga konfigurasi network, contract address, dan deployment dapat berbeda antara environment development dan production.

---

## 4.2. Verifikasi Audit Chain

Endpoint:

```http
GET /api/v1/audit/verify-chain
```

Endpoint ini digunakan untuk melakukan pengecekan integritas audit chain.

```bash
curl http://localhost:8080/api/v1/audit/verify-chain \
  -H "Authorization: Bearer $TOKEN"
```

Endpoint ini membutuhkan authentication.

---

## 4.3. Deteksi Data Tampering & Admin Security Alerts

Ketika staf rumah sakit mencoba mengakses data pasien (`GET /api/v1/hospital/access-requests/{id}/data`), sistem melakukan validasi kriptografis real-time dengan menghitung ulang hash profil pasien saat itu juga dan membandingkannya dengan hash yang tersimpan di blockchain / ledger audit (`VerifyProfileOnChain`).

Jika data profil di database terindikasi telah dimanipulasi atau diubah secara ilegal (*tampered*):
1. Pengembalian data medis pasien langsung **dibatalkan/diblokir** (HTTP 409 Conflict `patient_data_integrity_violation`).
2. Kejadian dicatat ke rantai audit sebagai event `INTEGRITY_VIOLATION_BLOCKED` lengkap dengan metadata forensik (hash asli vs hash saat ini, ID pasien, ID rumah sakit, alasan insiden).
3. Administrator platform dapat memonitor seluruh insiden pelanggaran integritas ini melalui endpoint:

```http
GET /api/v1/admin/security-alerts?limit=20&offset=0
```

Role: `ADMIN`

Contoh:
```bash
curl "http://localhost:8080/api/v1/admin/security-alerts?limit=10&offset=0" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

Response:
```json
{
  "total": 1,
  "alerts": [
    {
      "id": "c1a2b3c4-d5e6-7f8a-9b0c-1d2e3f4a5b6c",
      "entityType": "ACCESS_REQUEST",
      "entityId": "e2f3g4h5-i6j7-8k9l-0m1n-2o3p4q5r6s7t",
      "action": "INTEGRITY_VIOLATION_BLOCKED",
      "actorId": "a9b8c7d6-e5f4-3a2b-1c0d-9e8f7a6b5c4d",
      "payload": {
        "error": "hash_mismatch",
        "patientId": "11223344-5566-7788-9900-aabbccddeeff",
        "hospitalId": "55667788-9900-aabb-ccdd-eeff00112233",
        "currentHash": "a1b2c3d4e5f6...",
        "recordedHash": "d4e5f6a1b2c3...",
        "reason": "Real-time patient profile hash does not match anchored ledger hash"
      },
      "prevHash": "f1e2d3c4...",
      "hash": "b5a4c3d2...",
      "createdAt": "2026-10-04T14:32:00Z"
    }
  ]
}
```

---

# 5. Enkripsi Data Sensitif

Data tertentu tidak disimpan sebagai plaintext di PostgreSQL.

Saat ini field sensitif seperti:

* NIK
* nomor asuransi

diproses menggunakan encryption layer di:

```text
internal/crypto/
```

NIK menggunakan:

```text
AES-256-GCM
```

sebelum disimpan ke database.

Selain ciphertext NIK, backend juga menyimpan:

```text
nik_hash
```

yang digunakan untuk mencari uniqueness NIK tanpa harus melakukan dekripsi seluruh data.

---

## 5.1. Key yang Digunakan

Ada tiga key utama:

```env
FIELD_ENCRYPTION_KEY=...
NIK_HASH_KEY=...
AUDIT_HASH_KEY=...
```

Fungsinya berbeda:

| Key                    | Fungsi                                    |
| ---------------------- | ----------------------------------------- |
| `FIELD_ENCRYPTION_KEY` | Enkripsi/dekripsi field sensitif          |
| `NIK_HASH_KEY`         | HMAC/hash NIK untuk lookup dan uniqueness |
| `AUDIT_HASH_KEY`       | Hash untuk audit trail                    |

Jangan menggunakan satu key untuk semua fungsi.

Untuk production, key sebaiknya disimpan menggunakan secrets manager seperti Vault atau cloud secrets manager, bukan `.env` biasa.

---

# 6. Identity / KTP Verification

Akesa memiliki sistem identity verification untuk memverifikasi identitas pasien berdasarkan dokumen KTP.

Perlu dibedakan:

```text
Clerk Authentication
        │
        ▼
"Siapa yang sedang login?"
```

sedangkan:

```text
KTP Verification
        │
        ▼
"Apakah identitas pasien sesuai dengan dokumen identitas?"
```

Jadi berhasil login menggunakan Clerk **tidak berarti identitas KTP pasien sudah terverifikasi**.

---

## 6.1. Status Verification

Status yang digunakan:

| Status              | Keterangan                               |
| ------------------- | ---------------------------------------- |
| `PENDING`           | Verification baru dibuat                 |
| `PROCESSING`        | Verification sedang diproses             |
| `DOCUMENT_REJECTED` | Dokumen ditolak                          |
| `DATA_MISMATCH`     | Data dokumen tidak sesuai dengan profile |
| `LIVENESS_FAILED`   | Liveness check gagal                     |
| `FACE_MISMATCH`     | Face matching gagal                      |
| `MANUAL_REVIEW`     | Menunggu pemeriksaan manual              |
| `VERIFIED`          | Identitas berhasil diverifikasi          |
| `EXPIRED`           | Verification sudah tidak berlaku         |

### Catatan implementasi

Implementasi backend saat ini mendukung **manual review**.

Artinya status:

```text
MANUAL_REVIEW
      │
      ├── approve
      │      ↓
      │   VERIFIED
      │
      └── reject
             ↓
      DOCUMENT_REJECTED
```

Status seperti:

```text
PROCESSING
DATA_MISMATCH
LIVENESS_FAILED
FACE_MISMATCH
```

sudah disiapkan pada model untuk mendukung pengembangan identity verification yang lebih otomatis di tahap berikutnya.

**Jangan menganggap sistem saat ini sudah melakukan OCR KTP, liveness detection, atau face matching otomatis jika provider tersebut belum diintegrasikan.**

---

# 7. Endpoint Identity Verification

Semua endpoint berikut membutuhkan:

```http
Authorization: Bearer <clerk_jwt_token>
```

---

## 7.1. Membuat Verification

```http
POST /api/v1/patient/identity/verifications
```

Digunakan untuk membuat proses verification baru untuk pasien yang sedang login.

```bash
curl -X POST \
  http://localhost:8080/api/v1/patient/identity/verifications \
  -H "Authorization: Bearer $TOKEN"
```

Response berisi informasi verification seperti:

```json
{
  "id": "verification-uuid",
  "status": "PENDING",
  "documentType": "KTP"
}
```

---

## 7.2. Melihat Verification

```http
GET /api/v1/patient/identity/verifications/{id}
```

Contoh:

```bash
curl \
  http://localhost:8080/api/v1/patient/identity/verifications/<verificationId> \
  -H "Authorization: Bearer $TOKEN"
```

Pasien hanya boleh melihat verification miliknya sendiri.

---

## 7.3. Melihat Verification Terbaru

```http
GET /api/v1/patient/identity/verifications/latest
```

Endpoint ini digunakan mobile app untuk mengetahui status verification terbaru setelah aplikasi restart atau user login kembali.

```bash
curl \
  http://localhost:8080/api/v1/patient/identity/verifications/latest \
  -H "Authorization: Bearer $TOKEN"
```

Jika pasien belum pernah membuat verification, endpoint dapat mengembalikan:

```http
404 Not Found
```

Mobile app dapat memperlakukan kondisi tersebut sebagai:

```text
belum melakukan verification
```

---

## 7.4. Upload Dokumen KTP

```http
POST /api/v1/patient/identity/verifications/{id}/document
```

Endpoint ini digunakan untuk meng-upload dokumen KTP.

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/patient/identity/verifications/<verificationId>/document \
  -H "Authorization: Bearer $TOKEN" \
  -F "document=@ktp.jpg"
```

Format dokumen yang didukung oleh implementasi saat ini:

```text
.jpg
.png
```

Dokumen disimpan menggunakan storage key yang terpisah dari data profile.

Struktur penyimpanan:

```text
storage/
└── identity/
    └── <patientID>/
        └── <verificationID>/
            └── <uuid>.jpg
```

Folder storage identity **tidak boleh diekspos sebagai static/public directory**.

---

# 8. Admin Identity Verification

Admin dapat melakukan review terhadap verification pasien.

Endpoint admin menggunakan role:

```text
ADMIN
```

---

## 8.1. Approve Verification

```http
POST /api/v1/admin/identity-verifications/{id}/approve
```

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/admin/identity-verifications/<verificationId>/approve \
  -H "Authorization: Bearer $TOKEN"
```

Verification yang berada pada status:

```text
MANUAL_REVIEW
```

dapat diubah menjadi:

```text
VERIFIED
```

---

## 8.2. Reject Verification

```http
POST /api/v1/admin/identity-verifications/{id}/reject
```

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/admin/identity-verifications/<verificationId>/reject \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Dokumen KTP tidak dapat diverifikasi"
  }'
```

Verification yang ditolak akan memiliki status:

```text
DOCUMENT_REJECTED
```

Alur sederhananya:

```text
Patient
   │
   ▼
Create Verification
   │
   ▼
Upload KTP
   │
   ▼
MANUAL_REVIEW
   │
   ├───────────────┐
   ▼               ▼
APPROVE          REJECT
   │               │
   ▼               ▼
VERIFIED       DOCUMENT_REJECTED
```

---

# 9. Patient Profile

Sebelum menggunakan fitur pasien lainnya, user harus memiliki role:

```text
PATIENT
```

dan melakukan profile setup.

---

## 9.1. Create Profile

```http
POST /api/v1/patient/profile
```

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/patient/profile \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "fullName": "Budi Santoso",
    "nik": "3271000000000001",
    "dateOfBirth": "1995-06-12",
    "gender": "MALE",
    "phoneNumber": "081234567890",
    "address": "Jl. Merdeka No. 1, Jakarta"
  }'
```

Field utama:

```text
fullName
nik
dateOfBirth
gender
phoneNumber
address
```

Field tambahan:

```text
bloodType
drugAllergy
medicalHistory
insuranceNumber
emergencyContactName
emergencyContactPhone
```

`gender`:

```text
MALE
FEMALE
```

NIK harus terdiri dari:

```text
16 digit
```

---

## 9.2. Get Profile

```http
GET /api/v1/patient/profile
```

```bash
curl \
  http://localhost:8080/api/v1/patient/profile \
  -H "Authorization: Bearer $TOKEN"
```

---

## 9.3. Update Profile

```http
PUT /api/v1/patient/profile
```

Body menggunakan format profile yang sama seperti create profile.

```bash
curl -X PUT \
  http://localhost:8080/api/v1/patient/profile \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "fullName": "Budi Santoso",
    "nik": "3271000000000001",
    "dateOfBirth": "1995-06-12",
    "gender": "MALE",
    "phoneNumber": "081234567890",
    "address": "Jl. Merdeka No. 2, Jakarta"
  }'
```

Jika identity verification pasien sudah `VERIFIED`, field identitas yang berasal dari KTP harus diperlakukan sebagai data yang sudah terverifikasi.

Pada sisi mobile, field identitas dapat dibuat non-editable.

Backend juga perlu melakukan enforcement sehingga client tidak dapat mengubah data terverifikasi hanya dengan memodifikasi request HTTP.

---

# 10. User Synchronization

## `POST /api/v1/users/sync`

Setelah user pertama kali login/register menggunakan Clerk, mobile app harus melakukan sync ke backend.

```bash
curl -X POST \
  http://localhost:8080/api/v1/users/sync \
  -H "Authorization: Bearer $TOKEN"
```

User baru akan dibuat dengan role default:

```text
PATIENT
```

Endpoint bersifat idempotent.

Artinya aman dipanggil berkali-kali.

---

# 11. Current User

## `GET /api/v1/me`

Digunakan untuk mendapatkan informasi user yang sedang login.

```bash
curl \
  http://localhost:8080/api/v1/me \
  -H "Authorization: Bearer $TOKEN"
```

Endpoint ini berguna untuk mengetahui:

```text
internal user ID
Clerk user ID
role
status
```

---

# 12. Hospital & Registration Flow

## 12.1. `GET /api/v1/hospitals`

Menampilkan rumah sakit dengan status:

```text
ACTIVE
```

```bash
curl \
  http://localhost:8080/api/v1/hospitals \
  -H "Authorization: Bearer $TOKEN"
```

Endpoint ini digunakan pasien atau user yang sudah terdaftar untuk melihat rumah sakit yang tersedia.

---

## 12.2. `GET /api/v1/hospitals/invitations/validate`

Memvalidasi kode undangan registrasi rumah sakit sebelum form pendaftaran dibuka di sisi client.

Query parameter:

| Parameter | Tipe     | Keterangan                               |
| --------- | -------- | ---------------------------------------- |
| `key`     | `string` | Kode undangan (format: `AKESA-XXXX-XXXX`) |

Contoh:

```bash
curl "http://localhost:8080/api/v1/hospitals/invitations/validate?key=AKESA-7X9K-3B2M" \
  -H "Authorization: Bearer $TOKEN"
```

Response jika valid (HTTP 200 OK):

```json
{
  "valid": true,
  "targetEmail": "kontak@sardjito.co.id",
  "expiresAt": "2026-10-04T22:46:52Z"
}
```

Status error yang dapat terjadi:
* `400 Bad Request`: `key_query_param_required`
* `404 Not Found`: `invitation_not_found`
* `409 Conflict`: `invitation_already_used` (kode sudah pernah digunakan)
* `410 Gone`: `invitation_expired` (masa berlaku kode telah kedaluwarsa)

---

## 12.3. `POST /api/v1/hospitals/register`

Mengirim formulir pendaftaran rumah sakit menggunakan kode undangan yang valid.

Data pendaftaran akan masuk ke tabel staging `hospital_applications` dengan status awal:

```text
PENDING_REVIEW
```

Kode undangan otomatis ditandai terpakai (`is_used = true`). Role user tetap `PATIENT` sampai permohonan disetujui oleh Admin.

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/hospitals/register \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "keyCode": "AKESA-7X9K-3B2M",
    "name": "RSUP Dr. Sardjito",
    "address": "Jl. Kesehatan No. 1, Sleman, DIY",
    "phone": "0274-587333",
    "email": "kontak@sardjito.co.id",
    "registrationNumber": "3471012",
    "npwp": "01.234.567.8-901.000",
    "licenseDocumentUrl": "https://storage.akesa.id/docs/izin-sardjito.pdf",
    "picPosition": "Kepala Instalasi Rekam Medis",
    "picFullName": "dr. Budi Santoso"
  }'
```

Response (HTTP 201 Created):

```json
{
  "id": "8ed55912-0760-43ae-9ba4-bc2d2da285cd",
  "invitationId": "c4d3e2f1-...",
  "applicantUserId": "a1b2c3d4-...",
  "name": "RSUP Dr. Sardjito",
  "address": "Jl. Kesehatan No. 1, Sleman, DIY",
  "phone": "0274-587333",
  "email": "kontak@sardjito.co.id",
  "registrationNumber": "3471012",
  "npwp": "01.234.567.8-901.000",
  "licenseDocumentUrl": "https://storage.akesa.id/docs/izin-sardjito.pdf",
  "picFullName": "dr. Budi Santoso",
  "picPosition": "Kepala Instalasi Rekam Medis",
  "status": "PENDING_REVIEW",
  "adminNotes": "",
  "createdAt": "2026-10-01T22:48:19Z",
  "updatedAt": "2026-10-01T22:48:19Z"
}
```

---

## 12.4. `GET /api/v1/hospitals/my-application`

Melihat status formulir permohonan pendaftaran rumah sakit milik user yang sedang login, beserta catatan/alasan dari Admin jika ada.

Contoh:

```bash
curl \
  http://localhost:8080/api/v1/hospitals/my-application \
  -H "Authorization: Bearer $TOKEN"
```

Response menampilkan detail data `hospital_applications` beserta field `status` dan `adminNotes`.

---

## 12.5. `PUT /api/v1/hospitals/my-application`

Mengirim pembaruan data/dokumen ketika permohonan berada dalam status:

```text
REVISION_REQUIRED
```

Setelah pelamar berhasil mengirim revisi, status permohonan otomatis kembali menjadi:

```text
PENDING_REVIEW
```

sehingga Admin dapat meninjau ulang kelengkapan permohonan.

Contoh:

```bash
curl -X PUT \
  http://localhost:8080/api/v1/hospitals/my-application \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "RSUP Dr. Sardjito",
    "address": "Jl. Kesehatan No. 1, Sleman, DIY",
    "phone": "0274-587333",
    "email": "kontak@sardjito.co.id",
    "registrationNumber": "3471012",
    "npwp": "01.234.567.8-901.000",
    "licenseDocumentUrl": "https://storage.akesa.id/docs/izin-sardjito-revisi-hd.pdf",
    "picPosition": "Kepala Instalasi Rekam Medis",
    "picFullName": "dr. Budi Santoso"
  }'
```

Jika status saat ini bukan `REVISION_REQUIRED`, endpoint mengembalikan error:
* `409 Conflict`: `application_not_revision_required`

---

# 13. Patient Access Request

## `GET /api/v1/patient/access-requests`

Menampilkan request akses yang masuk ke pasien.

Status yang dapat ditemukan antara lain:

```text
PENDING
APPROVED
REJECTED
REVOKED
```

---

## `POST /api/v1/patient/access-requests/{id}/approve`

Menyetujui access request.

```bash
curl -X POST \
  http://localhost:8080/api/v1/patient/access-requests/<id>/approve \
  -H "Authorization: Bearer $TOKEN"
```

Request harus:

```text
PENDING
```

dan harus dimiliki oleh pasien yang sedang login.

---

## `POST /api/v1/patient/access-requests/{id}/reject`

Menolak request.

```bash
curl -X POST \
  http://localhost:8080/api/v1/patient/access-requests/<id>/reject \
  -H "Authorization: Bearer $TOKEN"
```

---

## `POST /api/v1/patient/access-requests/{id}/revoke`

Mencabut akses yang sebelumnya telah diberikan.

```bash
curl -X POST \
  http://localhost:8080/api/v1/patient/access-requests/<id>/revoke \
  -H "Authorization: Bearer $TOKEN"
```

Setelah revoke, rumah sakit tidak dapat lagi menggunakan request tersebut untuk mengambil data pasien.

---

# 14. Patient History

## `GET /api/v1/patient/history`

Menampilkan riwayat aktivitas pasien yang dicatat melalui audit trail secara komprehensif (timeline pembaruan profil, permohonan akses dari rumah sakit, persetujuan/penolakan akses, pencatatan data yang diakses, hingga peringatan insiden integritas data/tampering).

Role: `PATIENT`

Contoh:
```bash
curl http://localhost:8080/api/v1/patient/history \
  -H "Authorization: Bearer $TOKEN"
```

Response mengembalikan array dari timeline log audit yang berkaitan dengan pasien yang sedang login:

```json
[
  {
    "id": "c1a2b3c4-...",
    "entityType": "PATIENT_PROFILE",
    "entityId": "e2f3g4h5-...",
    "action": "PROFILE_UPDATED",
    "actorId": "a9b8c7d6-...",
    "payload": {
      "profileHash": "a1b2c3d4..."
    },
    "prevHash": "000000000...",
    "hash": "e5f6a7b8...",
    "createdAt": "2026-10-04T12:00:00Z"
  },
  {
    "id": "d2b3c4d5-...",
    "entityType": "ACCESS_REQUEST",
    "entityId": "f3g4h5i6-...",
    "action": "DATA_ACCESSED",
    "actorId": "b1c2d3e4-...",
    "payload": {
      "categories": ["IDENTITY", "MEDICAL_BASIC"],
      "dataHash": "a1b2c3d4...",
      "hospitalId": "h1i2j3k4-...",
      "patientId": "e2f3g4h5-..."
    },
    "prevHash": "e5f6a7b8...",
    "hash": "9a8b7c6d...",
    "createdAt": "2026-10-04T12:30:00Z"
  }
]
```

Event yang dapat muncul di timeline pasien antara lain:
- `PROFILE_UPDATED`: Pembaruan data profil dan penjangkaran hash baru ke audit chain.
- `ACCESS_REQUESTED`: Permintaan akses data baru yang dikirimkan oleh staf rumah sakit.
- `ACCESS_APPROVED`: Pasien menyetujui permohonan akses data tertentu.
- `ACCESS_REJECTED`: Pasien menolak permohonan akses data.
- `ACCESS_REVOKED`: Pasien mencabut izin akses yang sebelumnya aktif.
- `DATA_ACCESSED`: Rumah sakit telah membaca/mengambil data medis pasien sesuai kategori izin.
- `INTEGRITY_VIOLATION_BLOCKED`: Terjadi peringatan keamanan saat rumah sakit mencoba mengambil data namun terdeteksi ketidakcocokan hash (indikasi data di database telah dimanipulasi).

---

# 15. Hospital Staff

Hospital staff harus sudah dihubungkan dengan rumah sakit oleh admin.

---

## 15.1. Create Access Request

```http
POST /api/v1/hospital/access-requests
```

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/hospital/access-requests \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patientCode": "AKS-4F2A9C1D",
    "purpose": "Pendaftaran rawat jalan poli umum",
    "categories": ["IDENTITY", "CONTACT"]
  }'
```

Kategori data:

```text
IDENTITY
CONTACT
MEDICAL_BASIC
INSURANCE
EMERGENCY_CONTACT
```

Rumah sakit hanya meminta kategori data yang diperlukan.

---

## 15.2. List Access Request

```http
GET /api/v1/hospital/access-requests
```

Menampilkan request yang berkaitan dengan rumah sakit tempat staff tersebut terdaftar.

---

## 15.3. Get Patient Data

```http
GET /api/v1/hospital/access-requests/{id}/data
```

Data hanya dapat diambil jika:

```text
request.status == APPROVED
```

Contoh:

```bash
curl \
  http://localhost:8080/api/v1/hospital/access-requests/<id>/data \
  -H "Authorization: Bearer $TOKEN"
```

Response hanya berisi kategori data yang disetujui pasien.

Misalnya:

```json
{
  "patientCode": "AKS-4F2A9C1D",
  "fullName": "Budi Santoso",
  "nik": "3271000000000001",
  "dateOfBirth": "1995-06-12",
  "gender": "MALE"
}
```

Jika pasien tidak memberikan permission untuk kategori `CONTACT`, maka field seperti nomor telepon atau alamat tidak dikembalikan.

Setiap akses data dicatat ke audit trail.

---

# 16. Admin Hospital Management

Untuk development/testing, user dapat dipromosikan menjadi admin secara manual:

```sql
UPDATE users
SET role = 'ADMIN'
WHERE clerk_user_id = '<clerk_user_id>';
```

---

## `POST /api/v1/admin/hospitals`

Mendaftarkan rumah sakit.

Status awal:

```text
PENDING
```

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/admin/hospitals \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "RS Sehat Sentosa",
    "address": "Jl. Kesehatan No. 10",
    "phone": "0219876543",
    "email": "admin@rssehat.id",
    "registrationNumber": "RS-2026-001"
  }'
```

---

## `GET /api/v1/admin/hospitals`

Menampilkan seluruh rumah sakit tanpa memfilter status.

---

## `PATCH /api/v1/admin/hospitals/{id}/verify`

Mengubah rumah sakit:

```text
PENDING → VERIFIED
```

---

## `PATCH /api/v1/admin/hospitals/{id}/activate`

Mengaktifkan rumah sakit.

```text
VERIFIED → ACTIVE
```

Setelah `ACTIVE`, rumah sakit dapat muncul pada:

```text
GET /api/v1/hospitals
```

---

## `PATCH /api/v1/admin/hospitals/{id}/deactivate`

Menonaktifkan rumah sakit.

---

## `POST /api/v1/admin/hospitals/{id}/staff`

Menghubungkan user ke rumah sakit dan menjadikannya:

```text
HOSPITAL_STAFF
```

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/admin/hospitals/<hospitalId>/staff \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "b2c3d4e5-...",
    "fullName": "Siti Aminah",
    "position": "Petugas Pendaftaran"
  }'
```

`userId` merupakan ID internal pada database Akesa, bukan Clerk User ID.

---

## `POST /api/v1/admin/hospitals/invitations`

Menghasilkan kode undangan pendaftaran sekali pakai dan tautan magic link (deep link mobile) untuk calon perwakilan rumah sakit.

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/admin/hospitals/invitations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "kontak@sardjito.co.id",
    "expiresInHours": 72
  }'
```

Response (HTTP 201 Created):

```json
{
  "keyCode": "AKESA-7X9K-3B2M",
  "magicLink": "akesa://register-hospital?key=AKESA-7X9K-3B2M",
  "targetEmail": "kontak@sardjito.co.id",
  "expiresAt": "2026-10-04T22:46:52Z"
}
```

Jika `expiresInHours` tidak diisi atau `<= 0`, masa berlaku default adalah 72 jam (3 hari).

---

## 16.1. Alur Peninjauan Permohonan Rumah Sakit (Application Review)

Formulir yang dikirimkan calon perwakilan rumah sakit masuk ke tabel staging `hospital_applications` menggunakan native PostgreSQL enum `hospital_application_status`:

| Status              | Keterangan                                                              |
| ------------------- | ----------------------------------------------------------------------- |
| `PENDING_REVIEW`    | Permohonan baru disubmit, menunggu pemeriksaan oleh Admin               |
| `REVISION_REQUIRED` | Admin meminta perbaikan data/dokumen; pelamar dapat mengedit permohonan |
| `REJECTED`          | Permohonan ditolak permanen; kunci hangus dan form terkunci             |
| `APPROVED`          | Permohonan disetujui; RS aktif, akun staf dibuat, role dinaikkan        |

Siklus transisi status permohonan:

```text
       [Applicant Submit Form]
                  │
                  ▼
          PENDING_REVIEW ◄──────────────────────┐
                  │                             │
                  ├───► request-revision        │
                  │           │                 │
                  │           ▼                 │
                  │     REVISION_REQUIRED       │
                  │           │                 │
                  │     [Applicant Update PUT] ─┘
                  │
                  ├───► reject
                  │           │
                  │           ▼
                  │        REJECTED (Kunci hangus)
                  │
                  └───► approve
                              │
                              ▼
                           APPROVED
                              │
                              ├─► Salin data RS ke tabel `hospitals` (ACTIVE)
                              ├─► Catat relasi staf di tabel `hospital_staff`
                              └─► Promosikan role user menjadi `HOSPITAL_STAFF`
```

---

## `GET /api/v1/admin/hospital-applications`

Menampilkan daftar seluruh permohonan pendaftaran rumah sakit.

Query parameter (opsional):

| Parameter | Tipe     | Pilihan Nilai                                                  |
| --------- | -------- | -------------------------------------------------------------- |
| `status`  | `string` | `PENDING_REVIEW`, `REVISION_REQUIRED`, `REJECTED`, `APPROVED` |

Contoh:

```bash
curl "http://localhost:8080/api/v1/admin/hospital-applications?status=PENDING_REVIEW" \
  -H "Authorization: Bearer $TOKEN"
```

---

## `GET /api/v1/admin/hospital-applications/{id}`

Menampilkan detail satu formulir permohonan pendaftaran rumah sakit.

Contoh:

```bash
curl \
  http://localhost:8080/api/v1/admin/hospital-applications/<applicationId> \
  -H "Authorization: Bearer $TOKEN"
```

---

## `POST /api/v1/admin/hospital-applications/{id}/request-revision`

Meminta perbaikan data atau dokumen kepada pelamar dengan catatan instruksi wajib.

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/admin/hospital-applications/<applicationId>/request-revision \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "notes": "Dokumen surat izin operasional tidak terbaca jelas. Mohon upload ulang hasil scan resolusi tinggi."
  }'
```

Status permohonan akan berubah menjadi:

```text
REVISION_REQUIRED
```

---

## `POST /api/v1/admin/hospital-applications/{id}/reject`

Menolak permohonan pendaftaran rumah sakit secara permanen dengan alasan penolakan wajib.

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/admin/hospital-applications/<applicationId>/reject \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "notes": "Nomor izin operasional rumah sakit tidak terdaftar pada Kementerian Kesehatan."
  }'
```

Status permohonan akan berubah menjadi:

```text
REJECTED
```

Permohonan ini tidak dapat diedit kembali dan kode undangan dianggap hangus secara permanen.

---

## `POST /api/v1/admin/hospital-applications/{id}/approve`

Menyetujui pendaftaran rumah sakit.

Contoh:

```bash
curl -X POST \
  http://localhost:8080/api/v1/admin/hospital-applications/<applicationId>/approve \
  -H "Authorization: Bearer $TOKEN"
```

Dalam satu transaksi database atomik:
1. Status permohonan diubah menjadi `APPROVED`.
2. Data rumah sakit disalin ke tabel operasional `hospitals` dengan status `ACTIVE`.
3. User pelamar dicatat sebagai penanggung jawab rumah sakit pada tabel `hospital_staff`.
4. Role user pelamar dinaikkan dari `PATIENT` menjadi `HOSPITAL_STAFF`.

---

# 17. Patient QR Credential

QR digunakan sebagai credential untuk membantu proses identifikasi pasien.

QR **bukan permission data**.

Memiliki QR tidak otomatis memberikan rumah sakit akses ke data pasien.

---

## `GET /api/v1/patient/qr`

Mengambil QR credential pasien yang sedang login.

```bash
curl \
  http://localhost:8080/api/v1/patient/qr \
  -H "Authorization: Bearer $TOKEN"
```

Response:

```json
{
  "displayCode": "AKS-7F3A9C21",
  "qrPayload": "AKESA-QR-...",
  "isActive": true
}
```

| Field         | Tipe      | Keterangan                                          |
| ------------- | --------- | --------------------------------------------------- |
| `displayCode` | `string`  | Identifier QR yang dapat ditampilkan                |
| `qrPayload`   | `string`  | Payload yang digunakan untuk menghasilkan gambar QR |
| `isActive`    | `boolean` | Menunjukkan apakah credential masih aktif           |

Mobile app bertanggung jawab mengubah `qrPayload` menjadi gambar QR.

---

## `POST /api/v1/patient/qr/rotate`

Membuat credential QR baru dan menonaktifkan credential sebelumnya.

```bash
curl -X POST \
  http://localhost:8080/api/v1/patient/qr/rotate \
  -H "Authorization: Bearer $TOKEN"
```

Setelah rotation:

```text
QR lama → inactive
QR baru → active
```

QR sebaiknya hanya ditampilkan setelah user melakukan verifikasi keamanan perangkat seperti:

```text
Fingerprint
Face ID
PIN
Pattern
Password perangkat
```

Verifikasi tersebut dilakukan oleh mobile app, bukan backend endpoint QR.

---

# 18. Alur Utama Pasien

Alur dasar penggunaan Akesa:

```text
Patient Login/Register
        │
        ▼
POST /users/sync
        │
        ▼
POST /patient/profile
        │
        ▼
Create Identity Verification
        │
        ▼
Upload KTP
        │
        ▼
MANUAL_REVIEW
        │
        ├───────────────┐
        ▼               ▼
     APPROVE          REJECT
        │
        ▼
    VERIFIED
        │
        ▼
GET /patient/qr
        │
        ▼
Device Security Check
        │
        ▼
Show QR
        │
        ▼
Hospital Staff identifies patient
        │
        ▼
POST /hospital/access-requests
        │
        ▼
Patient receives request
        │
        ├───────────────┐
        ▼               ▼
     APPROVE          REJECT
        │
        ▼
Hospital accesses approved data
        │
        ▼
Audit Trail
```

Jika pasien ingin menghentikan akses:

```text
APPROVED
   │
   ▼
REVOKE
   │
   ▼
Hospital access blocked
```

Jika pasien ingin mengganti credential QR:

```text
POST /patient/qr/rotate
        │
        ▼
Old QR → inactive
New QR → active
```

---

# 19. Ringkasan Endpoint

## Public

| Method | Endpoint  | Keterangan   |
| ------ | --------- | ------------ |
| `GET`  | `/`       | Root API     |
| `GET`  | `/health` | Health check |

## Authenticated User

| Method | Endpoint                                 | Keterangan                            |
| ------ | ---------------------------------------- | ------------------------------------- |
| `POST` | `/api/v1/users/sync`                     | Sync Clerk user                       |
| `GET`  | `/api/v1/me`                             | Current user                          |
| `GET`  | `/api/v1/hospitals`                      | List active hospitals                 |
| `GET`  | `/api/v1/hospitals/invitations/validate` | Validasi kode undangan RS             |
| `POST` | `/api/v1/hospitals/register`             | Registrasi formulir RS via invitation |
| `GET`  | `/api/v1/hospitals/my-application`       | Cek status permohonan RS pelamar      |
| `PUT`  | `/api/v1/hospitals/my-application`       | Submit ulang revisi permohonan RS     |
| `GET`  | `/api/v1/audit/verify-chain`             | Verify audit chain                    |

## Patient

| Method | Endpoint                                               | Keterangan                   |
| ------ | ------------------------------------------------------ | ---------------------------- |
| `POST` | `/api/v1/patient/profile`                              | Create profile               |
| `GET`  | `/api/v1/patient/profile`                              | Get profile                  |
| `PUT`  | `/api/v1/patient/profile`                              | Update profile               |
| `POST` | `/api/v1/patient/identity/verifications`               | Create identity verification |
| `GET`  | `/api/v1/patient/identity/verifications/latest`        | Get latest verification      |
| `GET`  | `/api/v1/patient/identity/verifications/{id}`          | Get verification             |
| `POST` | `/api/v1/patient/identity/verifications/{id}/document` | Upload KTP                   |
| `GET`  | `/api/v1/patient/access-requests`                      | List access requests         |
| `POST` | `/api/v1/patient/access-requests/{id}/approve`         | Approve request              |
| `POST` | `/api/v1/patient/access-requests/{id}/reject`          | Reject request               |
| `POST` | `/api/v1/patient/access-requests/{id}/revoke`          | Revoke access                |
| `GET`  | `/api/v1/patient/history`                              | Patient history              |
| `GET`  | `/api/v1/patient/qr`                                   | Get QR credential            |
| `POST` | `/api/v1/patient/qr/rotate`                            | Rotate QR credential         |

## Hospital Staff

| Method | Endpoint                                     | Keterangan                   |
| ------ | -------------------------------------------- | ---------------------------- |
| `POST` | `/api/v1/hospital/access-requests`           | Create access request        |
| `GET`  | `/api/v1/hospital/access-requests`           | List hospital requests       |
| `GET`  | `/api/v1/hospital/access-requests/{id}/data` | Access approved patient data |

## Admin

| Method  | Endpoint                                                    | Keterangan                               |
| ------- | ----------------------------------------------------------- | ---------------------------------------- |
| `POST`  | `/api/v1/admin/hospitals`                                   | Create hospital                          |
| `GET`   | `/api/v1/admin/hospitals`                                   | List all hospitals                       |
| `PATCH` | `/api/v1/admin/hospitals/{id}/verify`                       | Verify hospital                          |
| `PATCH` | `/api/v1/admin/hospitals/{id}/activate`                     | Activate hospital                        |
| `PATCH` | `/api/v1/admin/hospitals/{id}/deactivate`                   | Deactivate hospital                      |
| `POST`  | `/api/v1/admin/hospitals/{id}/staff`                        | Assign hospital staff                    |
| `POST`  | `/api/v1/admin/hospitals/invitations`                       | Generate invitation key & magic link     |
| `GET`   | `/api/v1/admin/hospital-applications`                       | List permohonan pendaftaran RS           |
| `GET`   | `/api/v1/admin/hospital-applications/{id}`                  | Detail permohonan pendaftaran RS         |
| `POST`  | `/api/v1/admin/hospital-applications/{id}/request-revision` | Minta revisi permohonan pendaftaran RS   |
| `POST`  | `/api/v1/admin/hospital-applications/{id}/reject`           | Tolak permohonan pendaftaran RS          |
| `POST`  | `/api/v1/admin/hospital-applications/{id}/approve`          | Setujui permohonan RS & promosi role PIC |
| `POST`  | `/api/v1/admin/identity-verifications/{id}/approve`         | Approve KTP verification                 |
| `POST`  | `/api/v1/admin/identity-verifications/{id}/reject`          | Reject KTP verification                  |
| `GET`   | `/api/v1/admin/security-alerts`                             | List insiden pelanggaran integritas data (tampering alerts) |

---

# 20. Migration

Migration SQL berada di:

```text
backend/migrations/
```

Setiap migration harus memiliki:

```text
XXXXXX_description.up.sql
XXXXXX_description.down.sql
```

Nomor migration harus terus bertambah.

Contoh:

```text
000009_create_patient_qr_credentials.up.sql
000009_create_patient_qr_credentials.down.sql
```

Setelah migration sudah masuk ke `main`, **jangan mengubah migration lama**.

Jika ada perubahan schema, buat migration baru.

Contoh:

```text
000010_add_xxx.up.sql
000010_add_xxx.down.sql
```

---

# 21. Development Checklist

Sebelum membuat PR:

```bash
make fmt
go build ./...
go vet ./...
```

Jika ada test:

```bash
go test ./...
```

Pastikan:

* Tidak ada secret yang masuk Git
* `.env` tidak di-commit
* Migration memiliki `.up.sql` dan `.down.sql`
* Endpoint menggunakan authentication yang sesuai
* Endpoint admin menggunakan role check
* Ownership resource dicek di service layer
* Error menggunakan sentinel error
* `errors.Is(...)` digunakan untuk pengecekan error
* Data sensitif tidak masuk ke log
* Dokumen KTP tidak disimpan di public/static directory

---

# 22. Git Workflow

Gunakan satu branch untuk satu fitur/domain.

Contoh:

```text
feature/identity-verification
feature/hospital-crud
feature/access-request-flow
feature/patient-qr
feature/audit-blockchain
```

Sebelum membuat PR:

```bash
git status
git pull origin main
```

Kemudian pastikan branch sudah berisi perubahan terbaru dari `main`.

Hindari melakukan force push ke branch bersama kecuali sudah disepakati oleh tim.

---

# 23. Prinsip Penting Akesa

Beberapa prinsip yang harus dipertahankan ketika menambahkan fitur baru:

### 1. Authentication ≠ Identity Verification

Clerk menjawab:

```text
Siapa yang login?
```

KTP verification menjawab:

```text
Apakah identitas pasien sudah diverifikasi?
```

---

### 2. QR ≠ Permission

QR hanya digunakan untuk membantu identifikasi pasien.

```text
QR
 ≠
Access Permission
```

Akses data tetap menggunakan:

```text
Access Request
       ↓
Patient Approval
       ↓
Approved Data Access
```

---

### 3. Role ≠ Ownership

Role hanya menentukan apakah user memiliki jenis akses tertentu.

Tetap ---

## 24. Status Implementasi

Secara garis besar backend Akesa saat ini memiliki komponen:

```text
[✓] Clerk Authentication
[✓] User Synchronization
[✓] Role-based Authorization
[✓] Patient Profile
[✓] Hospital Management
[✓] Hospital Invitation Key & Magic Link
[✓] Hospital Application Staging & Admin Review (Revision / Reject / Approve)
[✓] Hospital Staff
[✓] Access Request
[✓] Patient Consent
[✓] Patient QR Credential
[✓] Identity Verification Flow
[✓] KTP Document Upload
[✓] Manual Identity Review
[✓] Sensitive Field Encryption
[✓] NIK Hashing
[✓] Audit Hash Chain
[✓] Audit Chain Integrity Verification
[✓] Admin Security Alerts & Tamper Incident Monitoring
[✓] Patient Access & Security Timeline
[✓] In-App Notification Center & Unread Badge Counter
[✓] Mobile Push Notification & Device Token Management (FCM-Ready)
[✓] Hybrid Blockchain PoC
```

Identity verification otomatis seperti:

```text
KTP OCR
Liveness Detection
Face Matching
Automatic Data Matching
```

masih bergantung pada implementasi provider/verification pipeline dan belum boleh dianggap aktif hanya karena field/status tersebut tersedia di database.

---

# 25. Prinsip Arsitektur

Secara keseluruhan:

```text
                         ┌──────────────┐
                         │    Clerk     │
                         │     Auth     │
                         └──────┬───────┘
                                │
                                ▼
┌──────────────┐        ┌───────────────┐        ┌──────────────────┐
│ Flutter App  │ ─────► │  Go Backend   │ ─────► │ Push Gateway/FCM │
└──────────────┘        └───────┬───────┘        └────────┬─────────┘
                                │                         │
             ┌──────────────────┼──────────────────┐      ▼
             │                  │                  │   Layar HP
             ▼                  ▼                  ▼  (Pop-up Alert)
       Patient Domain     Hospital Domain     Identity
             │                  │           Verification
             │                  │                  │
             └──────────────────┼──────────────────┘
                                │
                                ▼
                         Access / Consent
                                │
                                ▼
                         Notification Layer
                                │
                                ▼
                          Audit Service
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
               PostgreSQL             Blockchain
               Hash Chain                PoC
```

Tujuan akhirnya adalah menjaga agar data pasien tetap berada di bawah kontrol pasien, sementara rumah sakit hanya mendapatkan data yang memang diperlukan dan telah mendapatkan permission.

---

# 26. Sistem Notifikasi (In-App Center & Mobile Push)

Sistem notifikasi Akesa menggabungkan **In-App Notification Center** (penyimpanan riwayat di PostgreSQL) dan **Mobile Push Gateway** (pendaftaran token perangkat FCM untuk Android/iOS).

### 26.1. Skema Database (Migration `000013`)

* **`notifications`**:
  * `id`: UUID (Primary Key)
  * `user_id`: UUID (Foreign Key ke `users(id)`)
  * `title`: VARCHAR(255)
  * `message`: TEXT
  * `category`: `notification_category` (`SECURITY_ALERT`, `ACCESS_REQUEST`, `HOSPITAL_LIFECYCLE`, `SYSTEM`)
  * `severity`: `notification_severity` (`INFO`, `WARNING`, `CRITICAL`)
  * `is_read`: BOOLEAN (Default `false`)
  * `action_url`: VARCHAR(255) (Deep-link di aplikasi mobile/web)
  * `metadata`: JSONB
  * `created_at`, `read_at`: TIMESTAMPTZ

* **`user_device_tokens`**:
  * `id`: UUID
  * `user_id`: UUID (Foreign Key ke `users(id)`)
  * `token`: TEXT UNIQUE (Token unik registrasi FCM)
  * `platform`: VARCHAR(20) (`android`, `ios`)
  * `created_at`, `updated_at`: TIMESTAMPTZ

### 26.2. Endpoint API Notifikasi

Semua endpoint dilindungi autentikasi (`Bearer <token>`):

| Method | Endpoint | Keterangan |
| :--- | :--- | :--- |
| `POST` | `/api/v1/notifications/device-token` | Registrasi token FCM perangkat setelah login |
| `DELETE` | `/api/v1/notifications/device-token` | Hapus token FCM perangkat saat logout |
| `GET` | `/api/v1/notifications?unread_only=false&limit=20&offset=0` | Daftar riwayat notifikasi pengguna |
| `GET` | `/api/v1/notifications/unread-count` | Jumlah notifikasi belum dibaca (badge lonceng) |
| `PATCH` | `/api/v1/notifications/{id}/read` | Menandai satu notifikasi telah dibaca |
| `POST` | `/api/v1/notifications/read-all` | Menandai semua notifikasi telah dibaca |

### 26.3. Event Pemicu Otomatis (Event Triggers)

1. **Pelanggaran Integritas Data (`INTEGRITY_VIOLATION_BLOCKED`)**:
   * Pasien menerima peringatan kritis bahwa ada upaya akses yang dibatalkan karena hash tidak cocok dengan blockchain.
   * Admin menerima broadcast notifikasi insiden keamanan dengan action URL `/admin/security-alerts`.
2. **Permintaan Izin Akses (`ACCESS_REQUESTED`)**:
   * Pasien menerima notifikasi permintaan izin akses saat staf RS mengajukan permohonan.
3. **Persetujuan / Penolakan Akses (`ACCESS_APPROVED` / `ACCESS_REJECTED`)**:
   * Staf rumah sakit yang mengajukan permintaan menerima notifikasi hasil keputusan pasien.
��─────┘
                                │
             ┌──────────────────┼──────────────────┐
             │                  │                  │
             ▼                  ▼                  ▼
       Patient Domain     Hospital Domain     Identity
             │                  │              Verification
             │                  │                  │
             └──────────────────┼──────────────────┘
                                │
                                ▼
                         Access / Consent
                                │
                                ▼
                           Audit Service
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
               PostgreSQL             Blockchain
               Hash Chain                PoC
```

Tujuan akhirnya adalah menjaga agar data pasien tetap berada di bawah kontrol pasien, sementara rumah sakit hanya mendapatkan data yang memang diperlukan dan telah mendapatkan permission.
