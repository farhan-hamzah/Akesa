# Python Modern KTP OCR AI Service

Service AI modern berbasis **FastAPI** dan **RapidOCR (PP-OCRv4 ONNX Runtime)** yang sangat ringan (~15 MB) dan super cepat (< 300 ms).

## Keunggulan Dibandingkan Baseline Lama
| Parameter | Baseline Lama (Faster R-CNN) | Modern Engine (PP-OCRv4 ONNX) |
|---|---|---|
| **Ukuran Model** | ~2.1 GB (3 file .pth raksasa) | **~15 MB** (Hemat >98%) |
| **Kecepatan** | 3 - 4 detik | **< 300 ms** (10x lebih cepat) |
| **Field yang Dibaca** | Hanya NIK & Nama | **Lengkap: NIK, Nama, Tempat/Tgl Lahir, Kelamin, Alamat, RT/RW, Agama, Status, Pekerjaan, Kewarganegaraan** |
| **Kompilasi C++** | Wajib MSVC Build Tools | **Zero Compilation (Siap Pakai)** |
| **Kesiapan Deploy** | Sangat berat, rawan OOM | **Sangat ringan, ideal untuk Docker / Serverless** |

## Menjalankan Service
```bash
cd ai-service
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000
```
