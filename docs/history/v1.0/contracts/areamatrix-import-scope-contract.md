# AreaMatrix Import Scope Contract

## Purpose

本文定义 AreaFlow v0.1 AreaMatrix adapter 的只读导入深度和 artifact metadata 策略。它补充
[`v0.1-import-mirror-contract.md`](v0.1-import-mirror-contract.md)、
[`data-model-v0.1.md`](data-model-v0.1.md)、
[`artifact-backup-restore-contract.md`](artifact-backup-restore-contract.md)、
[`adapter-profile-boundary.md`](adapter-profile-boundary.md)、
[`areamatrix-dogfood-contract.md`](areamatrix-dogfood-contract.md) 和
[`../migration/areamatrix-workflow-migration.md`](../migrations/areamatrix-workflow-migration.md)。

核心规则：v0.1 import 是 metadata index，不是历史内容迁移。AreaFlow 可以读取 AreaMatrix 的 workflow
索引、状态 ledger、少量机器可解析文件和文件 metadata，但不能复制 prompt、日志、报告、diff、evidence
原文，也不能修改 AreaMatrix。

## Scope Vocabulary

本文区分两层：

```text
read envelope:
  v0.1 允许 adapter 读取的最大范围。后续可以在这个范围内增加 metadata coverage。

minimum import set:
  当前 v0.1 关闭必须稳定导入的最小文件、字段和 artifact metadata。
```

`read envelope` 不是实现承诺；`minimum import set` 才是 v0.1 gate evidence 的直接验收对象。

## Read Envelope

v0.1 允许读取：

```text
examples/areamatrix/areaflow.yaml or managed project areaflow.yaml
workflow/residuals/**
workflow/versions/*/residuals/**
workflow/versions/*/version.yaml
workflow/versions/*/discussion/**
workflow/versions/*/middle-layer/**
workflow/versions/*/changes/**
workflow/versions/*/plans/**
workflow/versions/*/drafts/**
workflow/versions/*/queue/**
workflow/versions/*/promotion*
workflow/versions/*/projection/**
workflow/versions/*/closeout/**
workflow/versions/*/execution/_shared/progress.json
workflow/templates/**
workflow/README.md
tasks/active/**
tasks/done/**
tasks/backlog/**
tasks/indexes/**
```

但 v0.1 对这些路径只能做以下动作：

```text
parse known YAML / JSON / Markdown index files
stat files for size and existence
hash explicitly selected source files
count directories or files for rough inventory
record source_path, sha256, size_bytes, content_type and metadata
```

## Minimum Import Set

当前 v0.1 必须稳定导入：

```text
workflow/residuals/residuals.yaml:
  parse global residual items.
  parse version_residuals links.

workflow/versions/<version>/residuals/residuals.yaml:
  parse version_status.
  parse version residual items.
  create workflow_version metadata from linked version, source path and hash.
  mark v1-mvp immutable.

workflow/versions/v-template/README.md:
  import v-template as template-only version when present.
  hash README and count version-local artifacts.

workflow/versions/v1-mvp/execution/_shared/progress.json:
  parse task count and status counts only.
  never rewrite progress.json.

tasks/active:
  count active task directories.

tasks/done:
  count done task directories.

tasks/backlog/prompts:
  count backlog prompt packages.
```

Current AreaMatrix v0.1 status summary must at least contain:

```text
project = areamatrix
tasks.active
tasks.done
tasks.backlog_packages
tasks.backlog_open
tasks.backlog_closed
v1_execution.total
v1_execution.done
v1_execution.status
residual_count
version_count
```

## Artifact Metadata Minimum Set

v0.1 must index these artifact metadata candidates when present:

```text
workflow/residuals/residuals.yaml
workflow/versions/v1-mvp/residuals/residuals.yaml
workflow/versions/v1-mvp/execution/_shared/progress.json
tasks/indexes/residuals.md
tasks/backlog/README.md
workflow/versions/v-template/README.md
```

For each imported artifact row, AreaFlow must store:

```text
project_id
workflow_version_id when inferable from path
run_id = import run id
artifact_type
storage_backend
uri
source_path
sha256
size_bytes
content_type
metadata jsonb
```

The current AreaMatrix v0.1 importer stores historical project files with:

```text
storage_backend = external_project
uri = source_path
```

`external_project` means AreaFlow has metadata and a path reference only. It does not own the bytes, cannot restore the
content, and cannot archive, delete, move, upload or rewrite the source file.

If a future adapter revision uses `project_reference` for managed-project files, it must preserve the same
metadata-only semantics and update restore / artifact integrity reports accordingly.

## Artifact Type Mapping

v0.1 artifact type mapping:

```text
progress:
  path ends with progress.json.

residual_index:
  path contains residual.

task_index:
  path contains backlog.

source_file:
  fallback for selected source-like files, for example v-template README.
```

Content type mapping:

```text
*.yaml / *.yml -> application/yaml
*.json -> application/json
*.md -> text/markdown
other -> application/octet-stream
```

## Explicit Non-imports

v0.1 must not import original content from:

```text
workflow/versions/*/execution/prompts/**
workflow/versions/*/execution/logs/**
workflow/versions/*/execution/reports/**
workflow/versions/*/execution/evidence/**
workflow/versions/*/execution/checkpoints/**
workflow/versions/*/execution/_shared/run_summaries/**
workflow/versions/*/drafts/**/*.md prompt bodies
workflow/versions/*/plans/**/*.md full plan bodies
workflow/versions/*/changes/**/*.yaml as authoritative product source
docs/** product source documents
source code
release evidence raw bundles
user files outside workflow/tasks indexes
```

These paths can be referenced by path/hash/size metadata only when explicitly selected by the adapter. They cannot be
copied into AreaFlow-owned artifact store in v0.1.

## Workflow Item Mapping

v0.1 maps AreaMatrix state to generic objects as follows:

```text
workflow_versions:
  one row per linked version residual ledger plus optional v-template.

workflow_items:
  only stage-level or residual-derived metadata that the adapter can identify without reading large raw content.

residuals:
  global residual items and version residual items.

artifacts:
  selected index/progress/template files as metadata-only external_project references.
```

v0.1 does not create:

```text
queue_candidate execution facts
live promotion mappings
run_task rows
run_attempt rows
approval records
gate results from AreaMatrix native commands
checkpoint facts
repair facts
```

If an imported residual has `executable_task=true`, it remains metadata. It cannot become a queue item or execution task
without later promotion, approval and execution gates.

## Hash And Drift Semantics

v0.1 `source_hash` means:

```text
file hash for selected source files
status summary hash for import snapshot
```

It does not prove:

```text
full workflow tree content unchanged
all execution logs indexed
all evidence copied
drift check completed
native doctor passed
```

v0.2 `import-diff` may compare repeated import snapshots, but v0.1 only has to produce enough snapshot data for that
later comparison.

## Safety Facts

Every real AreaMatrix import smoke must prove:

```text
project_write_attempted=false
execution_write_attempted=false
engine_call_attempted=false
task_loop_run_attempted=false
secret_resolved=false
status_json_unchanged=true unless explicit projection apply was authorized
workflow_readme_unchanged=true
protected_paths_clean=true
```

The protected path proof must include:

```bash
git -C /Users/as/Ai-Project/project/AreaMatrix status --short -- \
  workflow/README.md \
  .areaflow/status.json \
  scripts/task_loop/console.py \
  scripts/dev_tools/cli.py \
  scripts/task_loop/runner.py \
  scripts/areaflow_shim.py \
  workflow/versions \
  workflow/versions/v1-mvp/execution/_shared/progress.json
```

## Current Implementation Evidence

Current v0.1 evidence must remain consistent with:

```text
docs/development/areamatrix-adapter-import-evidence.md
docs/development/status-projection-evidence.md
```

At the latest recorded read-only smoke, AreaFlow imported six AreaMatrix artifact metadata rows, all with
`storage_backend=external_project`, all with `sha256`, `source_path` and `size_bytes`, and no non-external artifact rows.

## Anti-patterns

v0.1 import must reject these interpretations:

- “AreaFlow hashed a file, so it owns the file.”
- “`external_project` artifact exists, so restore is ready.”
- “progress.json was parsed, so execution state migrated.”
- “artifact count covers a stage, so all prompt/log/evidence original content was imported.”
- “residual `executable_task=true`, so a worker can execute it.”
- “repeated import snapshot means full drift doctor passed.”
- “v-template README was imported, so template apply is enabled.”
