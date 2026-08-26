# Akesa Backend

Ini backend-nya Akesa - aplikasi pendaftaran pasien antar-rumah sakit, dengan
audit trail berbasis hash chain (proof of concept "blockchain"-nya). Dibangun
pakai Go 1.25, PostgreSQL, dan Clerk buat autentikasi.

Dokumen ini isinya cara jalanin project-nya di lokal, gimana struktur kodenya
disusun, dan contoh pakai tiap endpoint biar kalau mau tes atau connect dari
Flutter gak perlu nebak-nebak.

## 1. Cara jalanin di lokal

```bash
cp .env.example .env          # isi CLERK_SECRET_KEY dari dashboard Clerk kamu
```

Sebelum lanjut, generate 3 key buat lapisan enkripsi NIK (dijelasin lebih
detail di section 4.5 di bawah), terus tempel hasilnya ke `.env`:

```bash
openssl rand -base64 32   # buat FIELD_ENCRYPTION_KEY
openssl rand -hex 32      # buat NIK_HASH_KEY
openssl rand -hex 32      # buat AUDIT_HASH_KEY (jangan sama dengan yang di atas!)
```

Baru lanjut:

```bash
docker compose up -d          # nyalain Postgres lokal (atau `make db-up` kalau punya make)
make migrate-up               # jalanin semua migrasi (butuh CLI golang-migrate)
make run                      # jalanin API di :8080
```

Kalau belum punya `golang-migrate`, install sekali aja:

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Terus cek API-nya udah hidup atau belum:

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

Kalau muncul itu, berarti udah beres. Lanjut baca bagian bawah buat paham
struktur kodenya sebelum mulai ngoprek.

## 2. Struktur kode

Tiap domain (`user`, `patient`, `hospital`, `access`, `audit`) bentuknya sama
persis, ada di `internal/<domain>/`:

| File            | Isinya apa                                                  |
|------------------|--------------------------------------------------------------|
| `model.go`      | struct domain-nya (bukan struct database, bukan struct JSON) |
| `errors.go`     | error yang udah didefinisiin (`var ErrX = errors.New(...)`)  |
| `repository.go` | tempat satu-satunya yang boleh nulis query SQL               |
| `dto.go`        | bentuk request JSON + validasi input                         |
| `service.go`    | logic bisnisnya, gak tau soal HTTP sama sekali             |
| `handler.go`    | urusan HTTP: baca JSON, panggil service, balikin response    |

Alurnya selalu satu arah, gak boleh lompat-lompat:

```
handler -> service -> repository -> Postgres
```

`cmd/api/main.go` itu satu-satunya tempat yang nyambungin semua layer ini
jadi satu. Jadi kalau mau nambah domain baru, yang perlu disentuh cuma: bikin
folder baru di `internal/<domain_baru>/`, tambahin satu blok wiring di
`main.go`, terus daftarin route-nya di `internal/server/server.go`. Udah,
gak perlu ubah-ubah domain lain.

### Kenapa banyak banget `interface` kecil-kecil?

Contohnya nih, package `patient` itu **gak** import package `audit` sama
sekali. Tapi di `patient/service.go` ada begini:

```go
type AuditRecorder interface {
    RecordProfileHash(ctx context.Context, patientID uuid.UUID, profileHash string) error
}
```

Kebetulan `audit.Service` punya method dengan tanda tangan yang sama persis,
jadi dia otomatis "cocok" sama interface itu tanpa `patient` perlu tau
package `audit` itu ada. Enaknya:

- gak bakal ada import cycle antar domain (bikin pusing kalau kejadian).
- Gampang di-unit-test - tinggal bikin mock kecil buat interface-nya, gak
  perlu nyalain Postgres beneran.
- Kalau suatu saat `audit` mau diganti pake blockchain sungguhan (misal
  Hyperledger), `patient` sama `access` gak perlu diapa-apain.

Kalau bingung liat pola ini di tempat lain, itu memang sengaja, bukan salah
ketik.

## 3. Auth & Role - WAJIB DIBACA sebelum nambah endpoint

- **Autentikasi** ditangani Clerk sepenuhnya. Mobile app kirim JWT Clerk di
  header `Authorization: Bearer <token>`, terus middleware
  `auth.RequireAuth` yang validasi.
- Tapi habis lolos autentikasi Clerk, itu **belum tentu** kita tau role-nya
  apa di sistem kita (`PATIENT` / `HOSPITAL_STAFF` / `ADMIN`). Role itu cuma
  ada di tabel `users` kita sendiri, bukan di token Clerk.
- **`auth.WithUser(userService)`** yang nyari baris di tabel `users`
  berdasarkan Clerk user id, terus nyimpen `{id, role, status}`-nya sebagai
  `AuthUser` di context request.
- **`auth.RequireRole("ADMIN")`** baca `AuthUser` itu dari context, kalau
  role-nya gak cocok ya di-reject (403).

Urutan middleware-nya **harus** selalu kayak gini, gak boleh dibolak-balik:

```go
auth.RequireAuth -> auth.WithUser(userService) -> auth.RequireRole(...)
```

Biar gak capek nulis 3 middleware tiap route, ada helper `chain` di
`internal/server/server.go`:

```go
s.mux.Handle("PATCH /api/v1/admin/hospitals/{id}/verify",
    s.chain(deps, http.HandlerFunc(deps.HospitalHandler.Verify), RoleAdmin),
)
```

> Dulu sempet ada bug di `RequireRole` - parameter role-nya gak dicek sama
> sekali, jadi siapa aja yang udah login bisa akses endpoint admin. Udah
> diperbaiki, sekarang role-nya beneran dicek dari database, bukan dari
> klaim yang dikirim client.

**Kalau mau nambah endpoint baru, tolong diinget:**
1. Jangan pernah percaya role/identitas dari body request. Selalu ambil dari
   `auth.GetAuthUser(r)` setelah middleware jalan.
2. Buat endpoint yang sifatnya milik satu user (misal approve access
   request), tetep cek kepemilikan di service layer (`req.PatientID ==
   authUser.ID`). Role doang gak cukup.
3. Endpoint yang khusus admin/staff wajib lewat `s.chain(..., RoleAdmin)`
   atau `s.chain(..., RoleStaff)`. Jangan taruh langsung di `mux.Handle`
   tanpa role check kalau memang harusnya dibatasi.

## 4. Soal audit trail ("blockchain"-nya)

`internal/audit` itu isinya hash chain di Postgres: tiap entri nyimpen
`sha256(prevHash + data)`, jadi urutan kejadian gak bisa diubah sepihak
tanpa ngerusak seluruh rantai setelahnya. Ini yang jadi "permissioned
blockchain proof of concept" yang disebut di SRS. **Data pribadi pasien
gak pernah masuk ke chain ini**, yang disimpen cuma hash-nya sama metadata
kecil (aksi apa, kapan, siapa).

Interface-nya sengaja dipisah biar kalau nanti tim mau ganti ke blockchain
permissioned beneran (misal Hyperledger Fabric), tinggal bikin implementasi
baru dengan method yang sama - `patient` sama `access` gak perlu diubah
sama sekali.

## 4.5. NIK - kenapa perlu diperlakukan beda dari field lain

NIK itu data paling sensitif di seluruh sistem ini, jadi diperlakukan beda
dari field lain kayak nama atau alamat. Ada dua masalah yang dipisahin, dan
dua-duanya udah ditangani:

**Masalah 1 - NIK kesimpen plain text di Postgres.**
Kalau database-nya suatu saat bocor/di-dump, NIK semua pasien ikut bocor
mentah-mentah. Solusinya: `internal/crypto` nyediain `FieldCipher` yang
ngenkripsi NIK (dan nomor asuransi) pake AES-256-GCM sebelum disimpen, dan
otomatis didekripsi lagi pas dibaca. Prosesnya transparan - service &
handler tetep kerja sama string biasa, cuma `patient/repository.go` aja
yang tau soal enkripsi ini. Yang kesimpen di kolom `nik` sekarang cuma
ciphertext acak, beda tiap kali di-enkripsi ulang biar dua orang dengan NIK
sama pun ciphertext-nya gak keliatan mirip.

**Masalah 2 - hash yang masuk ke audit chain bisa di-brute-force.**
Sebelumnya, hash yang dicatet ke audit chain itu `sha256(profil)` biasa.
Kedengeran aman, tapi NIK cuma 16 digit dengan format yang lumayan
terstruktur (kode wilayah + tanggal lahir + nomor urut), jadi ruang
kemungkinannya gak sebesar 16 digit acak - orang yang punya hash-nya bisa
nyoba brute-force offline nebak-nebak NIK sampe ketemu yang cocok.
Solusinya: pake `KeyedHasher` (HMAC-SHA256 dengan secret key) buat hitung
hash yang masuk ke chain, bukan SHA256 polos. Tanpa tau key-nya, brute-force
itu jadi gak feasible lagi.

Ada 3 key terpisah yang dipakai (lihat `.env.example`):
- `FIELD_ENCRYPTION_KEY` - buat enkripsi/dekripsi NIK & nomor asuransi.
- `NIK_HASH_KEY` - buat hash pencarian/keunikan NIK yang kesimpen di kolom
  `nik_hash` (biar sistem masih bisa cek "NIK ini udah kedaftar belum" tanpa
  perlu dekripsi semua baris satu-satu).
- `AUDIT_HASH_KEY` - buat hash yang masuk ke audit chain.

Sengaja dipisah tiga-tiganya (bukan satu key buat semua) - kalau salah satu
bocor entah gimana, yang lain masih aman. Di production, key-key ini
jangan pernah ditaruh di `.env` biasa - pake secrets manager (AWS Secrets
Manager, Vault, dsb) dan rotate key-nya secara berkala.

> Catatan buat yang udah sempet jalanin migrasi lama: migrasi
> `000007_encrypt_sensitive_fields` ngasumsiin tabel `patient_profiles`
> masih kosong. Kalau kamu udah sempet isi data dummy sebelum ini, hapus
> aja datanya dulu (atau reset volume Postgres-nya) sebelum jalanin migrasi
> ini, soalnya NIK yang kesimpen plain text sebelumnya gak bisa
> otomatis ke-enkripsi.

## 5. Semua endpoint & cara pakainya

Semua endpoint (kecuali `/health`) butuh header:

```
Authorization: Bearer <clerk_jwt_token>
```

Dan `POST`/`PUT` selalu `Content-Type: application/json`.

### Umum (siapa aja yang udah login)

#### `POST /api/v1/users/sync`
Wajib dipanggil **sekali** habis user pertama kali login/daftar lewat Clerk
- ini yang bikin baris di tabel `users` kita (role default-nya `PATIENT`).
Kalau belum pernah manggil ini, semua endpoint lain bakal nolak dengan
`user_not_registered`. Idempotent, jadi aman dipanggil berkali-kali, kalau
udah ada ya balikin yang udah ada aja.

```bash
curl -X POST http://localhost:8080/api/v1/users/sync \
  -H "Authorization: Bearer $TOKEN"
```

Response (201 kalau baru, 200 kalau udah ada):
```json
{
  "id": "a1b2c3d4-...",
  "clerkUserId": "user_2abc...",
  "role": "PATIENT",
  "status": "ACTIVE",
  "createdAt": "2026-08-26T10:00:00Z",
  "updatedAt": "2026-08-26T10:00:00Z"
}
```

#### `GET /api/v1/me`
Buat cek "login sebagai siapa sih, role-nya apa". Enak buat debugging
atau nentuin UI mana yang harus ditampilin di app.

```bash
curl http://localhost:8080/api/v1/me -H "Authorization: Bearer $TOKEN"
```

#### `GET /api/v1/hospitals`
List rumah sakit yang statusnya `ACTIVE` aja - buat pasien milih mau daftar
ke rumah sakit mana. gak perlu role khusus, yang penting udah pernah sync.

```bash
curl http://localhost:8080/api/v1/hospitals -H "Authorization: Bearer $TOKEN"
```

---

### Endpoint khusus PATIENT

Sebelum ini semua dipakai, user harus udah role `PATIENT` (default habis
sync) dan udah bikin profile dulu.

#### `POST /api/v1/patient/profile`
Isi data diri sekali aja - ini yang bikin `patientCode` unik yang nanti
dikasih ke petugas rumah sakit pas mau daftar.

```bash
curl -X POST http://localhost:8080/api/v1/patient/profile \
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

Field opsional yang bisa ditambahin: `bloodType`, `insuranceNumber`,
`emergencyContactName`, `emergencyContactPhone`. `gender` cuma nerima
`MALE` atau `FEMALE`, `nik` harus persis 16 digit, `dateOfBirth` formatnya
`YYYY-MM-DD`.

Response bakal ngasih `patientCode` (contoh: `AKS-4F2A9C1D`) - **catet ini**,
soalnya ini yang dipakai petugas RS buat cari data kamu.

NIK-nya sendiri disimpen terenkripsi di database (lihat section 4.5), tapi
di request/response API tetep string biasa kayak biasa - enkripsinya
kejadian di belakang layar, gak ngubah cara app manggil endpoint ini.
Kalau NIK yang dikirim udah kepake di profile lain, bakal kena `409
nik_already_registered`.

#### `GET /api/v1/patient/profile`
Liat profile sendiri.

#### `PUT /api/v1/patient/profile`
Update profile. Body-nya sama kayak `POST`, semua field wajib diisi ulang
(bukan partial update).

#### `GET /api/v1/patient/access-requests`
List semua permintaan akses yang pernah masuk ke kamu - baik yang masih
`PENDING`, udah `APPROVED`, `REJECTED`, atau `REVOKED`.

#### `POST /api/v1/patient/access-requests/{id}/approve`
Setujui permintaan akses. `{id}` diambil dari list di atas.

```bash
curl -X POST http://localhost:8080/api/v1/patient/access-requests/<id>/approve \
  -H "Authorization: Bearer $TOKEN"
```

Cuma bisa approve request yang statusnya masih `PENDING` dan itu request
punya kamu sendiri (dicek di server, bukan cuma dipercaya dari client).

#### `POST /api/v1/patient/access-requests/{id}/reject`
Sama kayak approve, tapi nolak. Sama-sama cuma bisa dari status `PENDING`.

#### `POST /api/v1/patient/access-requests/{id}/revoke`
Ini beda - buat nyabut akses yang **sebelumnya udah di-approve**. Bisa
kapan aja, gak ada batas waktu. Habis di-revoke, RS yang bersangkutan
langsung gak bisa lagi ambil data lewat request itu.

#### `GET /api/v1/patient/history`
Riwayat lengkap - semua perubahan profile, keputusan approve/reject/revoke,
sama kapan aja data kamu diakses RS. Ini yang narik dari audit hash chain.

---

### Endpoint khusus HOSPITAL_STAFF

User perlu udah dilink ke suatu rumah sakit sama admin dulu (lewat endpoint
admin di bawah) sebelum bisa pakai ini.

#### `POST /api/v1/hospital/access-requests`
Ajuin permintaan liat data pasien tertentu. `patientCode` didapet dari
pasiennya langsung (dikasih tau pas daftar).

```bash
curl -X POST http://localhost:8080/api/v1/hospital/access-requests \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patientCode": "AKS-4F2A9C1D",
    "purpose": "Pendaftaran rawat jalan poli umum",
    "categories": ["IDENTITY", "CONTACT"]
  }'
```

Kategori yang valid: `IDENTITY`, `CONTACT`, `MEDICAL_BASIC`, `INSURANCE`,
`EMERGENCY_CONTACT`. Minta secukupnya aja sesuai kebutuhan pendaftaran -
sistemnya emang didesain biar RS cuma minta kategori yang relevan, bukan
"kasih semua data lu".

#### `GET /api/v1/hospital/access-requests`
List semua request yang pernah diajukan RS kamu (bukan cuma yang kamu ajuin
sendiri, tapi semua staff di RS yang sama), plus status-nya masing-masing.

#### `GET /api/v1/hospital/access-requests/{id}/data`
Ambil data pasiennya - **cuma jalan kalau request-nya udah `APPROVED`**.
Kalau masih pending atau udah di-revoke, bakal ditolak.

```bash
curl http://localhost:8080/api/v1/hospital/access-requests/<id>/data \
  -H "Authorization: Bearer $TOKEN"
```

Response-nya cuma field yang sesuai kategori yang disetujui pasien, misal
kalau cuma disetujui `IDENTITY`:
```json
{
  "patientCode": "AKS-4F2A9C1D",
  "fullName": "Budi Santoso",
  "nik": "3271000000000001",
  "dateOfBirth": "1995-06-12",
  "gender": "MALE"
}
```
Nomor telepon, alamat, dst gak bakal ikut kebawa kalau kategori `CONTACT`
gak disetujui. Tiap kali endpoint ini dipanggil, otomatis kecatet di audit
log sebagai `DATA_ACCESSED`.

---

### Endpoint khusus ADMIN

Buat testing, cara paling gampang jadi admin: sync dulu biasa, terus update
manual di database:
```sql
UPDATE users SET role = 'ADMIN' WHERE clerk_user_id = '<clerk_user_id_kamu>';
```

#### `POST /api/v1/admin/hospitals`
Daftarin rumah sakit baru. Status awalnya otomatis `PENDING`.

```bash
curl -X POST http://localhost:8080/api/v1/admin/hospitals \
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

#### `GET /api/v1/admin/hospitals`
List semua rumah sakit, apapun statusnya (beda sama `GET /hospitals` yang
cuma nampilin yang `ACTIVE`).

#### `PATCH /api/v1/admin/hospitals/{id}/verify`
Ubah status dari `PENDING` ke `VERIFIED`.

#### `PATCH /api/v1/admin/hospitals/{id}/activate`
Ubah ke `ACTIVE` - abis ini baru rumah sakitnya nongol di `GET /hospitals`
dan staff-nya bisa mulai ngajuin access request.

#### `PATCH /api/v1/admin/hospitals/{id}/deactivate`
Nonaktifin RS. Semua tiga endpoint di atas bentuknya sama, cuma beda status
tujuannya, dan sama-sama gak butuh body.

```bash
curl -X PATCH http://localhost:8080/api/v1/admin/hospitals/<id>/activate \
  -H "Authorization: Bearer $TOKEN"
```

#### `POST /api/v1/admin/hospitals/{id}/staff`
Nge-link user yang udah sync ke suatu rumah sakit, sekalian promosiin
role-nya jadi `HOSPITAL_STAFF`. `userId` diambil dari `GET /me` punya user
yang mau dijadiin staff (bukan Clerk user id-nya, tapi `id` internal kita).

```bash
curl -X POST http://localhost:8080/api/v1/admin/hospitals/<hospitalId>/staff \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "b2c3d4e5-...",
    "fullName": "Siti Aminah",
    "position": "Petugas Pendaftaran"
  }'
```

## 6. Contoh alur lengkap, dari nol sampe selesai

1. Pasien sync + bikin profile → dapet `patientCode`.
2. Petugas RS (yang udah dilink admin) ajuin access request pake
   `patientCode` itu, sebutin `categories` yang dibutuhin.
3. Pasien buka `GET /patient/access-requests`, nemu request-nya, approve.
4. Petugas RS `GET /hospital/access-requests/{id}/data` - dapet data sesuai
   kategori yang disetujui.
5. Kapan pun pasien berubah pikiran, tinggal revoke, dan RS langsung gak
   bisa ambil data lagi lewat request itu.
6. Semua langkah di atas otomatis kecatet di audit trail, bisa diliat
   pasien lewat `GET /patient/history`.

## 7. Hal-hal yang perlu diinget kalau kerja bareng

- Satu branch per fitur/domain aja, misal `feature/hospital-crud`,
  `feature/access-request-flow`. Jangan digabung-gabung.
- Migrasi SQL selalu bikin sepasang `.up.sql` / `.down.sql`, nomor urut naik
  terus. Kalau migrasi udah di-merge ke main, jangan diedit lagi - bikin
  migrasi baru aja kalau mau ubah sesuatu.
- Sebelum bikin PR, minimal jalanin `make fmt`, terus pastiin
  `go build ./...` sama `go vet ./...` bersih, gak ada warning/error.
- Error di tiap domain selalu pake sentinel error (`var ErrX =
  errors.New(...)`) terus dicek pake `errors.Is(...)`. Jangan bandingin
  string pesan error-nya langsung, gampang salah kalau pesannya berubah.