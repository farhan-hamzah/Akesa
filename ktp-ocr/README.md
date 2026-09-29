# Indonesian KTP OCR / Information Extraction System (Modern Architecture)

Sistem ekstraksi data KTP Indonesia terstruktur berkinerja tinggi menggunakan arsitektur microservices dua service: **Go Backend (Gin Gateway)** dan **Python AI Service (FastAPI + PP-OCRv4 ONNX Runtime)**.

---

## 1. Arsitektur & Keunggulan Modern

Sistem ini telah dimodernisasi dari baseline usang (Faster R-CNN seberat 2.1 GB) menjadi arsitektur modern berbasis **ONNX Runtime (PP-OCRv4)** yang siap produksi:

| Metrik | Model Baseline Lama | Arsitektur Modern Baru |
|---|---|---|
| **Ukuran Model** | ~2.100 MB (2.1 GB) | **~15 MB** (Hemat >98%) |
| **Kecepatan Inference** | ~3.000 - 4.000 ms | **< 300 ms** |
| **Cakupan Ekstraksi** | Hanya NIK & Nama | **Lengkap: Seluruh Field KTP** |
| **Kebutuhan Memori** | RAM >3 GB (sering OOM) | RAM <350 MB |
| **Dependensi Build** | Wajib C++ MSVC compiler | **Murni ONNX / Python standar** |

---

## 2. Arsitektur Service

```text
Client (cURL / Postman / Frontend)
           │
           │ HTTP multipart/form-data (field: "image")
           ▼
┌──────────────────────────────────────┐
│       Go Gin Backend (:8080)         │
│  - Validasi file (JPG/JPEG/PNG)      │
│  - Pembatasan ukuran (maks 15MB)     │
│  - In-memory stream forwarding       │
│  - Timeout pemanggilan terkontrol    │
│  - Privasi: Tanpa logging data KTP   │
└──────────────────┬───────────────────┘
                   │
                   │ HTTP multipart/form-data
                   ▼
┌──────────────────────────────────────┐
│     Python FastAPI Service (:8000)   │
│  - Model di-load sekali saat startup │
│  - PP-OCRv4 Text Detection & Recog   │
│  - KTP Field Pattern Matching        │
│  - NIK Wilayah & Date Fallback       │
│  - Output: Full Structured JSON      │
└──────────────────────────────────────┘
```

---

## 3. Struktur Folder

```text
ktp-ocr/
│
├── backend/
│   ├── main.go                       # Entrypoint Gin server (:8080)
│   ├── go.mod                        # Go module
│   ├── go.sum
│   ├── internal/
│   │   ├── client/
│   │   │   └── ai_client.go          # HTTP client pemanggil FastAPI
│   │   ├── service/
│   │   │   └── ocr_service.go        # Validasi file & orkestrasi
│   │   └── handler/
│   │       └── ocr_handler.go        # Gin REST controller
│   └── README.md
│
├── ai-service/
│   ├── app/
│   │   ├── main.py                   # FastAPI app dengan lifespan startup
│   │   ├── api/
│   │   │   └── endpoints.py          # Route POST /ocr & GET /health
│   │   ├── services/
│   │   │   └── ocr_service.py        # Pipeline ekstraksi KTP
│   │   ├── models/
│   │   │   └── detector.py           # Modern PP-OCRv4 ONNX Engine (~15MB)
│   │   ├── preprocessing/
│   │   │   └── image.py              # Decode & validasi gambar
│   │   └── ocr/
│   │       └── parser.py             # Parser seluruh field KTP + wilayah lookup
│   │
│   ├── weights/
│   │   └── data_kode_wilayah.csv     # Database kode wilayah Kemendagri
│   │
│   ├── requirements.txt
│   └── README.md
│
├── test-images/
│   └── sample.jpg                    # Foto sampel KTP untuk testing
│
├── README.md
└── .gitignore
```

---

## 4. Cara Menjalankan

### Terminal 1 — Jalankan Python AI Service (:8000)
```powershell
cd ktp-ocr/ai-service
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
```

### Terminal 2 — Jalankan Go Backend (:8080)
```powershell
cd ktp-ocr/backend
go run .
```

---

## 5. Cara Melakukan Inference (Testing)

### Test Langsung ke Python AI Service (:8000):
```powershell
curl.exe -s -X POST http://localhost:8000/ocr -F "image=@ktp-ocr/test-images/sample.jpg"
```

### Test End-to-End Melalui Go Gateway (:8080):
```powershell
curl.exe -s -X POST http://localhost:8080/api/v1/ktp/ocr -F "image=@ktp-ocr/test-images/sample.jpg"
```

### Format Response Lengkap:
```json
{
  "success": true,
  "data": {
    "nik": "3101021210920002",
    "nama": "YAZIDUL FAHMI",
    "tempat_lahir": "JAKARTA",
    "tanggal_lahir": "12-10-1992",
    "jenis_kelamin": "LAKI-LAKI",
    "alamat": "PULAU TIDUNG, RT/RW 005/002, Kel. PULAU TIDUNG, Kec. KEPULAUAN SERIBU SELATAN",
    "agama": "ISLAM",
    "status_perkawinan": "BELUM KAWIN",
    "pekerjaan": "PELAJAR/MAHASISWA",
    "kewarganegaraan": "WNI"
  }
}
```
