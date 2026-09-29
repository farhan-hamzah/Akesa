from typing import Dict, Any, Optional
from app.models.detector import ModernKTPOCREngine
from app.ocr.parser import ModernKTPParser
from app.preprocessing.image import validate_image_bytes, decode_image_bytes

class OCRService:
    def __init__(self, engine: ModernKTPOCREngine, parser: ModernKTPParser):
        self.engine = engine
        self.parser = parser

    def process_ktp_image(self, image_bytes: bytes, filename: str) -> Dict[str, Any]:
        # 1. Validation
        validate_image_bytes(image_bytes, filename)

        # 2. Decoding
        img_bgr = decode_image_bytes(image_bytes)

        # 3. Text Detection & Recognition via PP-OCRv4 ONNX
        ocr_results = self.engine.extract_text(img_bgr)

        # 4. KTP Structured Field Parsing
        ktp_data = self.parser.extract_fields(ocr_results)

        # Privasi: tidak melakukan logging data pribadi (NIK, Nama, dll.)
        return ktp_data
