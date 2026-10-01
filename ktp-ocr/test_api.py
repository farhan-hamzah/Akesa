import os
import sys
import requests
import json
import time

# Default test image: prioritize the latest uploaded image
DEFAULT_DIR = os.path.join(os.path.dirname(__file__), "test-images")
USER_SCREENSHOT = os.path.join(DEFAULT_DIR, "Screenshot 2026-09-29 214305.png")
SAMPLE_IMAGE = os.path.join(DEFAULT_DIR, "sample.jpg")

if len(sys.argv) > 1 and os.path.exists(sys.argv[1]):
    TARGET_IMAGE = sys.argv[1]
elif os.path.exists(USER_SCREENSHOT):
    TARGET_IMAGE = USER_SCREENSHOT
else:
    TARGET_IMAGE = SAMPLE_IMAGE

def test_python_service(image_path: str):
    print("\n" + "=" * 60)
    print(f"[1] TESTING PYTHON AI SERVICE (Direct http://localhost:8000/ocr)")
    print(f"Target Image: {os.path.basename(image_path)}")
    print("=" * 60)
    url = "http://localhost:8000/ocr"

    try:
        t0 = time.time()
        with open(image_path, "rb") as f:
            files = {"image": (os.path.basename(image_path), f, "image/png")}
            resp = requests.post(url, files=files, timeout=30)
        dur = (time.time() - t0) * 1000

        print(f"Status Code : {resp.status_code}")
        print(f"Latency     : {dur:.2f} ms")
        print("Response JSON:")
        print(json.dumps(resp.json(), indent=2))
    except requests.exceptions.ConnectionError:
        print("[GAGAL] Tidak dapat terhubung ke Python AI Service di port 8000.")
        print("Pastikan kamu sudah menjalankan di terminal 1:\n  cd ai-service\n  python -m uvicorn app.main:app --host 0.0.0.0 --port 8000")

def test_go_backend(image_path: str):
    print("\n" + "=" * 60)
    print(f"[2] TESTING GO BACKEND GATEWAY (http://localhost:8080/api/v1/ktp/ocr)")
    print(f"Target Image: {os.path.basename(image_path)}")
    print("=" * 60)
    url = "http://localhost:8080/api/v1/ktp/ocr"

    try:
        t0 = time.time()
        with open(image_path, "rb") as f:
            files = {"image": (os.path.basename(image_path), f, "image/png")}
            resp = requests.post(url, files=files, timeout=30)
        dur = (time.time() - t0) * 1000

        print(f"Status Code : {resp.status_code}")
        print(f"Latency     : {dur:.2f} ms")
        print("Response JSON:")
        print(json.dumps(resp.json(), indent=2))
    except requests.exceptions.ConnectionError:
        print("[GAGAL] Tidak dapat terhubung ke Go Backend di port 8080.")
        print("Pastikan kamu sudah menjalankan di terminal 2:\n  cd backend\n  go run .")

if __name__ == "__main__":
    print(f"Memulai pengujian dengan gambar: {TARGET_IMAGE}")
    test_python_service(TARGET_IMAGE)
    test_go_backend(TARGET_IMAGE)
