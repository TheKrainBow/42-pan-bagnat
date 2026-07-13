UPDATE magic_link_tokens
   SET consumed_at = NOW()
 WHERE consumed_at IS NULL
   AND (token_value IS NULL OR token_value = '');
