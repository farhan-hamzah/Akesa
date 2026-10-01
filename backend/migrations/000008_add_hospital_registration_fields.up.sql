ALTER TABLE hospitals 
ADD COLUMN npwp VARCHAR(30),
ADD COLUMN license_document_url TEXT,
ADD COLUMN pic_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_hospitals_pic_user_id ON hospitals(pic_user_id);
