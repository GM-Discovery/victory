-- Kernel 96: this used to unconditionally create a second location named
-- 'victory-theater' on every database, redundant with (and blind to)
-- whatever slug an install's Operator actually chose. Fully superseded by
-- access.EnsureDefaultLocation (Go), which now runs between the schema
-- migrations (000-001) and this one, creating exactly one is_default
-- location under the install's real DEFAULT_LOCATION_SLUG. Left as a
-- harmless no-op rather than removed -- migration history is append-only.
SELECT 1;
