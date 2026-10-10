DROP INDEX IF EXISTS idx_hospitals_pic_user_id;

ALTER TABLE hospitals 
DROP COLUMN IF EXISTS npwp,
DROP COLUMN IF EXISTS license_document_url,
DROP COLUMN IF EXISTS pic_user_id;
