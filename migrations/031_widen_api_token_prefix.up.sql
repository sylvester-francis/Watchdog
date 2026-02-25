-- H-022: increase API token prefix column width from 11 to 20 characters.
-- New tokens store 16-char prefixes; the extra 4 chars of headroom avoids
-- another migration if the prefix grows again.
ALTER TABLE api_tokens ALTER COLUMN prefix TYPE VARCHAR(20);
