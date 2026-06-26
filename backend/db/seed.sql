-- Demo data for Healthwatch: ~125 watched sites with a realistic up/down mix.
--
-- Statuses are NOT stored here: the scheduler re-checks every URL within 30s,
-- so the URLs below are chosen to settle fast. Real, reliable sites (spread
-- across many domains so no single host is hammered by the 30s sweep) resolve
-- "up" quickly; the *.invalid hosts fail DNS in milliseconds and settle "down"
-- - no slow 5s timeouts that would saturate a small VM the way a flood of
-- identical example.com items did.
--
-- Run with: task seed   (clears existing items first, so it is idempotent)

DELETE FROM items;

-- ~95 "up": real, fast, CDN-backed sites, round-robined across 15 domains.
INSERT INTO items (name, url)
SELECT
  (ARRAY[
    'Acme Corp', 'Globex', 'Initech', 'Umbrella', 'Hooli',
    'Stark Industries', 'Wayne Enterprises', 'Wonka', 'Cyberdyne', 'Soylent',
    'Pied Piper', 'Vandelay', 'Massive Dynamic', 'Aperture', 'Tyrell'
  ])[1 + (g % 15)] || ' #' || g AS name,
  (ARRAY[
    'https://www.google.com', 'https://github.com', 'https://www.cloudflare.com',
    'https://www.wikipedia.org', 'https://www.mozilla.org', 'https://www.amazon.com',
    'https://www.microsoft.com', 'https://www.apple.com', 'https://about.gitlab.com',
    'https://www.debian.org', 'https://www.kernel.org', 'https://www.gnu.org',
    'https://nginx.org', 'https://www.postgresql.org', 'https://www.python.org'
  ])[1 + (g % 15)] AS url
FROM generate_series(1, 95) AS g;

-- ~30 "down": hostnames under the reserved .invalid TLD never resolve, so the
-- checker marks them down almost instantly.
INSERT INTO items (name, url)
SELECT
  'Legacy node ' || g AS name,
  'https://node-' || g || '.invalid' AS url
FROM generate_series(1, 30) AS g;
