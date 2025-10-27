-- Remove severity column from knowledge_articles table
ALTER TABLE knowledge_articles DROP COLUMN IF EXISTS severity;
