BEGIN;

UPDATE productions
SET name = 'Main Production'
WHERE slug = 'kernel-7-production'
  AND name = 'Kernel 7 Production';

COMMIT;
