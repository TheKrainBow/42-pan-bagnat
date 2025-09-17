-- Strip "-<branch>" suffix from module slugs, matching the sanitized branch format
-- 1) Compute sanitized branch (lowercase, non-alnum -> '-', trim '-') per module
WITH branch AS (
  SELECT id,
         btrim(regexp_replace(lower(git_branch), '[^a-z0-9]+', '-', 'g'), '-') AS branch_slug
    FROM modules
)
UPDATE modules m
   SET slug = regexp_replace(m.slug, '-' || b.branch_slug || '$', '')
  FROM branch b
 WHERE m.id = b.id
   AND b.branch_slug <> ''
   AND m.slug ~ ('-' || b.branch_slug || '$');

-- 2) Resolve duplicates by appending -<last4 of id> to all but the first per slug
WITH dups AS (
  SELECT slug, array_agg(id ORDER BY id) AS ids
    FROM modules
   GROUP BY slug
  HAVING COUNT(*) > 1
), tofix AS (
  SELECT unnest(ids[2:]) AS id
    FROM dups
)
UPDATE modules m
   SET slug = m.slug || '-' || right(m.id, 4)
  FROM tofix
 WHERE m.id = tofix.id;

