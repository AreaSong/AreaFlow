# AreaFlow Tasks

`tasks/**` 只用于仓库内尚未进入 AreaFlow 自托管流程的轻量开发记录。AreaFlow 产品中的 project、workflow、run 和 task 状态以 PostgreSQL 为源事实，不从本目录推导运行状态。

- `active/`：当前人工维护的仓库任务。
- `backlog/`：尚未批准执行的候选工作。
- `done/`：已完成的轻量任务记录。
- `templates/`：任务记录模板。

历史 0-100 平台 backlog 已归档到 [`../docs/history/v1.0/plans/task-backlog.md`](../docs/history/v1.0/plans/task-backlog.md)。未来产品方向见 [`../docs/roadmap.md`](../docs/roadmap.md)。
