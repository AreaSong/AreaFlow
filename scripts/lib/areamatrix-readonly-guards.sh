#!/usr/bin/env bash

areaflow_file_fingerprint() {
  local path="$1"

  if [[ ! -e "${path}" ]]; then
    echo "__missing__"
    return
  fi

  local stat_value
  if stat_value="$(stat -f '%m:%z' "${path}" 2>/dev/null)"; then
    :
  else
    stat_value="$(stat -c '%Y:%s' "${path}")"
  fi

  local hash
  hash="$(shasum -a 256 "${path}" | awk '{print $1}')"
  echo "${stat_value}:${hash}"
}

areaflow_protected_path_git_status() {
  local project_root="$1"
  shift

  git -C "${project_root}" status --short -- "$@"
}

areaflow_protected_path_status_hash() {
  local status_output="$1"

  printf "%s" "${status_output}" | shasum -a 256 | awk '{print $1}'
}

areaflow_protected_path_fingerprint() {
  local project_root="$1"
  local target_uri="$2"
  shift 2

  python3 - "${project_root}" "${target_uri}" "$@" <<'PY'
import hashlib
import os
import stat
import sys

root = os.path.abspath(sys.argv[1])
target = os.path.abspath(os.path.join(root, sys.argv[2]))
protected_paths = sys.argv[3:]


def rel(path):
    return os.path.relpath(path, root).replace(os.sep, "/")


def entry(path):
    info = os.lstat(path)
    mode = info.st_mode
    relative = rel(path)
    if stat.S_ISREG(mode):
        with open(path, "rb") as handle:
            content = handle.read()
        return f"{relative}\tfile\t{len(content)}\t{hashlib.sha256(content).hexdigest()}"
    if stat.S_ISDIR(mode):
        return f"{relative}\tdir"
    if stat.S_ISLNK(mode):
        return f"{relative}\tsymlink\t{os.readlink(path)}"
    return f"{relative}\tother\t{stat.filemode(mode)}\t{info.st_size}"


def walk(path, entries):
    entries.append(entry(path))
    if not stat.S_ISDIR(os.lstat(path).st_mode):
        return
    for name in sorted(os.listdir(path)):
        walk(os.path.join(path, name), entries)


entries = []
for protected_path in protected_paths:
    absolute = os.path.abspath(os.path.join(root, protected_path))
    if absolute == target:
        continue
    try:
        os.lstat(absolute)
    except FileNotFoundError:
        entries.append(f"{rel(absolute)}\tmissing")
        continue
    walk(absolute, entries)

payload = "\n".join(entries).encode()
print(hashlib.sha256(payload).hexdigest())
PY
}

areaflow_readonly_side_effect_counts() {
  local database_url="$1"
  local project_key="$2"

  psql "${database_url}" -v "project_key=${project_key}" -At <<'SQL'
WITH project_scoped_tables AS (
    SELECT table_schema, table_name
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND column_name = 'project_id'
      AND table_name <> 'projects'
    GROUP BY table_schema, table_name
),
nullable_project_tables AS (
    SELECT table_schema, table_name
    FROM information_schema.columns
    WHERE table_schema = 'public'
      AND column_name = 'project_id'
      AND is_nullable = 'YES'
      AND table_name <> 'projects'
    GROUP BY table_schema, table_name
),
global_tables(table_name) AS (
    VALUES
      ('actors'),
      ('users'),
      ('teams'),
      ('memberships'),
      ('adapters'),
      ('workflow_profiles'),
      ('schema_migrations'),
      ('migration_ledger')
),
existing_global_tables AS (
    SELECT gt.table_name
    FROM global_tables gt
    JOIN information_schema.tables t
      ON t.table_schema = 'public'
     AND t.table_name = gt.table_name
     AND t.table_type = 'BASE TABLE'
),
dynamic_query AS (
    SELECT
      'SELECT ''project.projects'' AS scope, COUNT(*)::bigint AS row_count, COALESCE(md5(string_agg(row_payload, E''\n'' ORDER BY row_payload)), '''') AS row_hash FROM (SELECT to_jsonb(p)::text AS row_payload FROM public.projects p WHERE p.project_key = '
      || quote_literal(:'project_key')
      || ') scoped_rows'
      || COALESCE(
        ' UNION ALL ' || (
          SELECT string_agg(
            format(
              'SELECT %L AS scope, COUNT(*)::bigint AS row_count, COALESCE(md5(string_agg(row_payload, E''\n'' ORDER BY row_payload)), '''') AS row_hash FROM (SELECT to_jsonb(t)::text AS row_payload FROM %I.%I t JOIN public.projects p ON p.id = t.project_id WHERE p.project_key = %L) scoped_rows',
              'project.' || table_name,
              table_schema,
              table_name,
              :'project_key'
            ),
            ' UNION ALL '
          )
          FROM project_scoped_tables
        ),
        ''
      )
      || COALESCE(
        ' UNION ALL ' || (
          SELECT string_agg(
            format(
              'SELECT %L AS scope, COUNT(*)::bigint AS row_count, COALESCE(md5(string_agg(row_payload, E''\n'' ORDER BY row_payload)), '''') AS row_hash FROM (SELECT to_jsonb(t)::text AS row_payload FROM %I.%I t WHERE t.project_id IS NULL) scoped_rows',
              'global_null_project.' || table_name,
              table_schema,
              table_name
            ),
            ' UNION ALL '
          )
          FROM nullable_project_tables
        ),
        ''
      )
      || COALESCE(
        ' UNION ALL ' || (
          SELECT string_agg(
            format(
              'SELECT %L AS scope, COUNT(*)::bigint AS row_count, COALESCE(md5(string_agg(row_payload, E''\n'' ORDER BY row_payload)), '''') AS row_hash FROM (SELECT to_jsonb(t)::text AS row_payload FROM public.%I t) scoped_rows',
              'global.' || table_name,
              table_name
            ),
            ' UNION ALL '
          )
          FROM existing_global_tables
        ),
        ''
      )
      || ' ORDER BY scope' AS sql
)
SELECT sql FROM dynamic_query;
\gexec
SQL
}
