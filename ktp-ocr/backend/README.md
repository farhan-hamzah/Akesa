# Go Backend Service - KTP OCR API Gateway

Service backend berbasis **Go (Golang)** dan **Gin Framework** yang bertindak sebagai API gateway untuk sistem OCR KTP Indonesia.

## Fitur & Tanggung Jawab
- **Validasi Gambar**: Memvalidasi format file (hanya JPG, JPEG, PNG) dan ukuran file (maksimal 15MB).
- **Streaming Forwarding**: Meneruskan payload gambar via `multipart/form-data` ke Python AI Service tanpa menyimpan file secara permanen ke disk.
- **Resilient Client**: Menggunakan HTTP Client dengan timeout terkontrol (default 60 detik).
- **Privasi**: Tidak melakukan logging data sensitif (NIK, Nama, tanggal lahir).
- **CORS Support**: Mengizinkan akses client dari domain mana saja.

## Konfigurasi Environment Variables
| Variable | Default | Keterangan |
|---|---|---|
| `PORT` | `8080` | Port listen HTTP server Go |
| `AI_SERVICE_URL` | `http://localhost:8000` | URL service Python FastAPI |
| `TIMEOUT_SECONDS` | `60` | Timeout pemanggilan ke AI service |

## Menjalankan Service
```bash
cd backend
go run .
```
atau menggunakan binary hasil build:
```bash
go build -o ktp-backend.exe .
./ktp-backend.exe
```
