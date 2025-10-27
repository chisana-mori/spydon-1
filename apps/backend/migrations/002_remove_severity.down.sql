-- Add back severity column to knowledge_articles table (for rollback)
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS severity VARCHAR(16);
