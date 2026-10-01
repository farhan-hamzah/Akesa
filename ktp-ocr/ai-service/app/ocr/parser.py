import os
import re
import pandas as pd
from typing import Dict, Any, List, Optional

class ModernKTPParser:
    def __init__(self, wilayah_csv_path: Optional[str] = None):
        self.wilayah_df: Optional[pd.DataFrame] = None
        if wilayah_csv_path and os.path.exists(wilayah_csv_path):
            try:
                self.wilayah_df = pd.read_csv(wilayah_csv_path)
                if "KodeWilayah" in self.wilayah_df.columns:
                    self.wilayah_df["KodeWilayah"] = self.wilayah_df["KodeWilayah"].astype(str)
            except Exception as e:
                print(f"[WARN] Gagal membaca database kode wilayah: {e}")

    def clean_value(self, val: str) -> str:
        if not val:
            return ""
        # Remove leading colons, dashes, dots, spaces
        cleaned = re.sub(r"^[\s:：;\-\.]+", "", val).strip()
        # Collapse multiple spaces
        cleaned = re.sub(r"\s+", " ", cleaned).strip()
        return cleaned

    def extract_fields(self, ocr_results: List[Any]) -> Dict[str, Optional[str]]:
        data: Dict[str, Optional[str]] = {
            "nik": None,
            "nama": None,
            "tempat_lahir": None,
            "tanggal_lahir": None,
            "jenis_kelamin": None,
            "alamat": None,
            "agama": None,
            "status_perkawinan": None,
            "pekerjaan": None,
            "kewarganegaraan": None
        }

        if not ocr_results:
            return data

        lines = [item[1].strip() for item in ocr_results if len(item) >= 2 and item[1]]
        full_text = "\n".join(lines)

        # 1. NIK Extraction (16 digits)
        nik_match = re.search(r"\b(\d{16})\b", full_text)
        if nik_match:
            data["nik"] = nik_match.group(1)
        else:
            for i, line in enumerate(lines):
                line_no_space = re.sub(r"\s+", "", line).upper()
                if "NIK" in line_no_space:
                    for offset in [0, 1, 2]:
                        if i + offset < len(lines):
                            digits = re.sub(r"\D", "", lines[i+offset])
                            if len(digits) >= 15:
                                data["nik"] = digits[:16]
                                break
                    if data["nik"]:
                        break

        # 2. Iterate lines for specific labeled fields
        alamat_street = None
        alamat_rt_rw = None
        alamat_kel = None
        alamat_kec = None

        for i, line in enumerate(lines):
            line_str = line.strip()
            line_upper = line_str.upper()

            # NAMA
            if "NAMA" in line_upper and not data["nama"]:
                parts = re.split(r"[:：]", line_str, maxsplit=1)
                if len(parts) > 1 and parts[1].strip():
                    data["nama"] = self.clean_value(parts[1])
                elif i + 1 < len(lines):
                    cand = self.clean_value(lines[i+1])
                    if not any(k in cand.upper() for k in ["TEMPAT", "LAHIR", "JENIS", "ALAMAT"]):
                        data["nama"] = cand

            # TEMPAT / TGL LAHIR
            if ("TEMPAT" in line_upper or "LAHIR" in line_upper) and (not data["tempat_lahir"] or not data["tanggal_lahir"]):
                val = line_str
                if ":" in val or "：" in val:
                    val = re.split(r"[:：]", val, maxsplit=1)[1]
                val = self.clean_value(val)

                tgl_match = re.search(r"(\d{2}[-/]\d{2}[-/]\d{4})", val)
                if tgl_match:
                    data["tanggal_lahir"] = tgl_match.group(1).replace("/", "-")
                    tempat = val[:tgl_match.start()].rstrip(", -")
                    data["tempat_lahir"] = self.clean_value(tempat) if tempat else None
                elif i + 1 < len(lines):
                    next_line = lines[i+1]
                    next_tgl = re.search(r"(\d{2}[-/]\d{2}[-/]\d{4})", next_line)
                    if next_tgl:
                        data["tanggal_lahir"] = next_tgl.group(1).replace("/", "-")
                        tempat = next_line[:next_tgl.start()].rstrip(", -")
                        data["tempat_lahir"] = self.clean_value(tempat) if tempat else None

            # JENIS KELAMIN
            if ("JENIS" in line_upper or "KELAMIN" in line_upper) and not data["jenis_kelamin"]:
                val_to_check = line_upper
                if i + 1 < len(lines):
                    val_to_check += " " + lines[i+1].upper()
                if "LAKI" in val_to_check:
                    data["jenis_kelamin"] = "LAKI-LAKI"
                elif "PEREMPUAN" in val_to_check:
                    data["jenis_kelamin"] = "PEREMPUAN"

            # AGAMA
            if "AGAMA" in line_upper and not data["agama"]:
                val_to_check = line_upper
                if i + 1 < len(lines):
                    val_to_check += " " + lines[i+1].upper()
                for ag in ["ISLAM", "KRISTEN", "KATOLIK", "HINDU", "BUDDHA", "KONGHUCU"]:
                    if ag in val_to_check:
                        data["agama"] = ag
                        break

            # STATUS PERKAWINAN
            if ("STATUS" in line_upper or "PERKAWINAN" in line_upper) and not data["status_perkawinan"]:
                val_to_check = line_upper
                if i + 1 < len(lines):
                    val_to_check += " " + lines[i+1].upper()
                if "BELUM" in val_to_check:
                    data["status_perkawinan"] = "BELUM KAWIN"
                elif "CERAI HIDUP" in val_to_check:
                    data["status_perkawinan"] = "CERAI HIDUP"
                elif "CERAI MATI" in val_to_check:
                    data["status_perkawinan"] = "CERAI MATI"
                elif "KAWIN" in val_to_check:
                    data["status_perkawinan"] = "KAWIN"

            # PEKERJAAN
            if "PEKERJAAN" in line_upper and not data["pekerjaan"]:
                parts = re.split(r"[:：]", line_str, maxsplit=1)
                if len(parts) > 1 and parts[1].strip():
                    data["pekerjaan"] = self.clean_value(parts[1])
                elif i + 1 < len(lines):
                    cand = self.clean_value(lines[i+1])
                    if not any(k in cand.upper() for k in ["KEWARGANEGARAAN", "BERLAKU", "WNI"]):
                        data["pekerjaan"] = cand

            # KEWARGANEGARAAN
            if ("KEWARGANEGARAAN" in line_upper or "WARGA" in line_upper) and not data["kewarganegaraan"]:
                val_to_check = line_upper
                for off in range(1, 4):
                    if i + off < len(lines):
                        val_to_check += " " + lines[i+off].upper()
                if "WNA" in val_to_check:
                    data["kewarganegaraan"] = "WNA"
                elif "WNI" in val_to_check or "WNE" in val_to_check or "INDONESIA" in val_to_check:
                    data["kewarganegaraan"] = "WNI"

            # ALAMAT DETAILS
            if "ALAMAT" in line_upper and not alamat_street:
                parts = re.split(r"[:：]", line_str, maxsplit=1)
                if len(parts) > 1 and parts[1].strip():
                    alamat_street = self.clean_value(parts[1])
                elif i + 1 < len(lines):
                    cand = self.clean_value(lines[i+1])
                    if not any(k in cand.upper() for k in ["RT", "RW", "KEL", "DESA", "KEC"]):
                        alamat_street = cand

            if ("RT/RW" in line_upper or ("RT" in line_upper and "RW" in line_upper)) and not alamat_rt_rw:
                parts = re.split(r"[:：]", line_str, maxsplit=1)
                val = self.clean_value(parts[1]) if len(parts) > 1 else self.clean_value(lines[min(i+1, len(lines)-1)])
                if val:
                    alamat_rt_rw = f"RT/RW {val}"

            if ("KEL/DESA" in line_upper or "KELDESA" in line_upper) and not alamat_kel:
                parts = re.split(r"[:：]", line_str, maxsplit=1)
                val = self.clean_value(parts[1]) if len(parts) > 1 else self.clean_value(lines[min(i+1, len(lines)-1)])
                if val:
                    alamat_kel = f"Kel. {val}"

            if "KECAMATAN" in line_upper and not alamat_kec:
                parts = re.split(r"[:：]", line_str, maxsplit=1)
                val = self.clean_value(parts[1]) if len(parts) > 1 else self.clean_value(lines[min(i+1, len(lines)-1)])
                if val:
                    alamat_kec = f"Kec. {val}"

        # Combine Alamat
        alamat_components = [c for c in [alamat_street, alamat_rt_rw, alamat_kel, alamat_kec] if c]
        if alamat_components:
            data["alamat"] = ", ".join(alamat_components)

        # 3. Fallback / Cross-Validation via NIK (if certain fields are missing)
        if data["nik"] and len(data["nik"]) == 16:
            clean_nik = data["nik"]
            # Wilayah lookup
            if not data["tempat_lahir"] and self.wilayah_df is not None:
                kode_wil = clean_nik[:6]
                match = self.wilayah_df[self.wilayah_df["KodeWilayah"] == kode_wil]
                if not match.empty:
                    row = match.iloc[0]
                    data["tempat_lahir"] = str(row.get("DaerahTingkatDua", ""))

            # Birthdate & Gender fallback
            if not data["tanggal_lahir"] or not data["jenis_kelamin"]:
                try:
                    day = int(clean_nik[6:8])
                    month = clean_nik[8:10]
                    year = clean_nik[10:12]
                    full_year = f"19{year}" if int(year) > 30 else f"20{year}"
                    if 31 < day < 72:
                        if not data["jenis_kelamin"]:
                            data["jenis_kelamin"] = "PEREMPUAN"
                        if not data["tanggal_lahir"]:
                            data["tanggal_lahir"] = f"{day - 40:02d}-{month}-{full_year}"
                    elif 1 <= day <= 31:
                        if not data["jenis_kelamin"]:
                            data["jenis_kelamin"] = "LAKI-LAKI"
                        if not data["tanggal_lahir"]:
                            data["tanggal_lahir"] = f"{day:02d}-{month}-{full_year}"
                except Exception:
                    pass

        return data
