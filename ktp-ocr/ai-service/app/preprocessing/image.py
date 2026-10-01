import io
import cv2
import numpy as np
from PIL import Image
from typing import Tuple, Optional

ALLOWED_EXTENSIONS = {"jpg", "jpeg", "png"}
MAX_FILE_SIZE = 15 * 1024 * 1024  # 15 MB

def validate_image_bytes(image_bytes: bytes, filename: str) -> None:
    if len(image_bytes) == 0:
        raise ValueError("File gambar kosong (0 bytes).")
    if len(image_bytes) > MAX_FILE_SIZE:
        raise ValueError(f"Ukuran file melebihi batas maksimum 15MB: {len(image_bytes)/(1024*1024):.2f}MB")
    
    ext = filename.split(".")[-1].lower() if "." in filename else ""
    if ext not in ALLOWED_EXTENSIONS:
        raise ValueError(f"Format file tidak didukung: .{ext}. Format yang diperbolehkan: jpg, jpeg, png.")

def decode_image_bytes(image_bytes: bytes) -> np.ndarray:
    nparr = np.frombuffer(image_bytes, np.uint8)
    img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)
    if img is None:
        raise ValueError("Gagal membaca file gambar. Format gambar rusak atau tidak valid.")
    return img

def preprocess_crop_for_char_detection(crop_bgr: np.ndarray, target_height: int = 100) -> Tuple[np.ndarray, int, int]:
    h, w = crop_bgr.shape[:2]
    if h == 0 or w == 0:
        raise ValueError("Crop gambar kosong.")
    
    target_width = max(10, int(w * target_height / h))
    resized = cv2.resize(crop_bgr, (target_width, target_height))
    
    # Convert BGR -> RGB -> Grayscale -> RGB -> BGR (as baseline model training mapper)
    pil_img = Image.fromarray(resized[:, :, ::-1]).convert("L").convert("RGB")
    bgr_normalized = np.asarray(pil_img)[:, :, ::-1]
    
    return bgr_normalized, target_height, target_width
