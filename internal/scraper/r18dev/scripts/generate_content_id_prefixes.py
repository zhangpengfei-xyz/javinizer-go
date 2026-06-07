#!/usr/bin/env python3
"""Generate content_id_prefixes.go from an r18.dev database dump.

Usage:
    python3 generate_content_id_prefixes.py <dump.sql> <output.go>

Download the dump from: https://r18.dev/dumps/latest
It redirects to an S3 URL like: https://r18dotdev.s3.eu-west-1.wasabisys.com/dumps/r18dotdev_dump_YYYY-MM-DD.sql.gz

Steps:
    1. curl -Lo r18dev_dump.sql.gz "https://r18.dev/dumps/latest"
    2. gunzip r18dev_dump.sql.gz
    3. python3 generate_content_id_prefixes.py r18dev_dump.sql ../content_id_prefixes.go
"""

import re
import sys
from collections import defaultdict


def extract_videos(sql_path):
    """Extract (content_id, dvd_id) pairs from the COPY block for derived_video."""
    rows = []
    in_copy = False
    with open(sql_path) as f:
        for line in f:
            line = line.rstrip('\n')
            if line.startswith('COPY public.derived_video '):
                in_copy = True
                continue
            if in_copy:
                if line == '\\.':
                    break
                parts = line.split('\t')
                if len(parts) >= 2:
                    rows.append((parts[0], parts[1]))
    return rows


def main():
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <dump.sql> <output.go>")
        sys.exit(1)

    sql_path = sys.argv[1]
    output_path = sys.argv[2]

    rows = extract_videos(sql_path)
    print(f"Extracted {len(rows)} video entries")

    content_id_pattern = re.compile(r'^(\d*)([a-z]+)(\d+)$')

    # Build: series -> set of prefixes from ALL rows
    series_prefixes = defaultdict(set)
    for cid, did in rows:
        m = content_id_pattern.match(cid)
        if not m:
            continue
        prefix = m.group(1)
        series = m.group(2)
        series_prefixes[series].add(prefix)

    # Sort prefixes: empty string first, then by length, then numerically
    def prefix_sort_key(p):
        if p == '':
            return (0, 0, '')
        return (1, len(p), int(p))

    # Generate Go source file
    lines = []
    lines.append('package r18dev')
    lines.append('')
    lines.append('//go:generate python3 scripts/generate_content_id_prefixes.py /tmp/r18dev_dump.sql content_id_prefixes.go')
    lines.append('')
    lines.append('// contentIDPrefixLookup maps series names (lowercase) to their known DMM content_id prefixes.')
    lines.append('// Built from r18.dev database dump. Regenerate with: go generate ./internal/scraper/r18dev/...')
    lines.append('var contentIDPrefixLookup = map[string][]string{')

    for series in sorted(series_prefixes.keys()):
        prefixes = sorted(series_prefixes[series], key=prefix_sort_key)
        prefix_strs = ', '.join(f'"{p}"' for p in prefixes)
        lines.append(f'\t"{series}": {{{prefix_str}}},')

    lines.append('}')
    lines.append('')

    with open(output_path, 'w') as f:
        f.write('\n'.join(lines))

    import os
    print(f"Wrote {len(series_prefixes)} series -> {sum(len(v) for v in series_prefixes.values())} prefix entries")
    print(f"Output: {output_path} ({os.path.getsize(output_path) / 1024:.1f} KB)")


if __name__ == '__main__':
    main()
