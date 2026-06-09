-- Add images column to announcements (array of image URLs)
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS images TEXT[] DEFAULT '{}';

COMMENT ON COLUMN announcements.images IS 'Array of uploaded image URLs for the announcement';
