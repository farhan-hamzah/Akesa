import os
import sys
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

# Ensure ai-service root is in sys.path
CURRENT_DIR = os.path.dirname(os.path.abspath(__file__))
AI_SERVICE_DIR = os.path.dirname(CURRENT_DIR)
if AI_SERVICE_DIR not in sys.path:
    sys.path.insert(0, AI_SERVICE_DIR)

from app.models.detector import ModernKTPOCREngine
from app.ocr.parser import ModernKTPParser
from app.services.ocr_service import OCRService
from app.api.endpoints import router as ocr_router

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: Load lightweight modern OCR model once into memory
    weights_dir = os.getenv("WEIGHTS_DIR", os.path.join(AI_SERVICE_DIR, "weights"))
    use_cuda = os.getenv("USE_CUDA", "false").lower() in ("true", "1", "yes")

    print("[STARTUP] Inisialisasi Modern KTP OCR Engine (PP-OCRv4 ONNX)...")
    engine = ModernKTPOCREngine(use_cuda=use_cuda)
    
    wilayah_csv = os.path.join(weights_dir, "data_kode_wilayah.csv")
    parser = ModernKTPParser(wilayah_csv_path=wilayah_csv)
    ocr_service = OCRService(engine=engine, parser=parser)

    app.state.engine = engine
    app.state.parser = parser
    app.state.ocr_service = ocr_service
    print("[STARTUP] Modern KTP OCR AI Service siap melayani request!")

    yield

    # Shutdown
    print("[SHUTDOWN] Membersihkan resource AI Service...")

app = FastAPI(
    title="Indonesian KTP OCR Modern AI Service",
    description="High-performance lightweight (~15MB) Computer Vision Service for Indonesian KTP Information Extraction",
    version="2.0.0",
    lifespan=lifespan
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(ocr_router)

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8000"))
    host = os.getenv("HOST", "0.0.0.0")
    uvicorn.run("app.main:app", host=host, port=port, reload=False)
