-- MVP-7 A1: agent display flag (display-only, no business logic may branch on it)
ALTER TABLE trait ADD COLUMN ai INTEGER DEFAULT 0;
