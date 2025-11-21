-- Re-add title column to knowledge_articles table (for rollback)
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS title VARCHAR(500) NOT NULL DEFAULT '';
