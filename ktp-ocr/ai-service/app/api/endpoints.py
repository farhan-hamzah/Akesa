from fastapi import APIRouter, UploadFile, File, HTTPException, Request
from fastapi.responses import JSONResponse
import time

router = APIRouter()

@router.get("/health")
async def health_check(request: Request):
    engine = getattr(request.app.state, "engine", None)
    return {
        "status": "healthy",
        "service": "ktp-ocr-ai-service",
        "device": getattr(engine, "device", "unknown") if engine else "not_loaded",
        "cuda_available": getattr(engine, "cuda_available", False) if engine else False
    }

@router.post("/ocr")
async def extract_ktp_ocr(request: Request, image: UploadFile = File(...)):
    ocr_service = getattr(request.app.state, "ocr_service", None)
    if ocr_service is None:
        raise HTTPException(status_code=503, detail="AI Service belum siap (model masih di-load).")

    try:
        image_bytes = await image.read()
        filename = image.filename or "unknown.jpg"
        
        result_data = ocr_service.process_ktp_image(image_bytes, filename)
        
        return {
            "success": True,
            "data": result_data
        }
    except ValueError as ve:
        return JSONResponse(
            status_code=400,
            content={"success": False, "error": str(ve)}
        )
    except Exception as e:
        return JSONResponse(
            status_code=500,
            content={"success": False, "error": f"Gagal memproses gambar: {str(e)}"}
        )
