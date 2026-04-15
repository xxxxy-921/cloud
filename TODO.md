openspec/changes/
  ├── itsm-form-engine/           ① 表单引擎 (独立, 无依赖)
  │   ├── .openspec.yaml
  │   └── proposal.md
  ├── itsm-process-variables/     ② 流程变量 (依赖 ①)
  │   ├── .openspec.yaml
  │   └── proposal.md
  ├── itsm-execution-tokens/      ③ 执行令牌 (依赖 ②)
  │   ├── .openspec.yaml
  │   └── proposal.md
  ├── itsm-gateway-parallel/      ④ 并行/包含网关 (依赖 ③)
  │   ├── .openspec.yaml
  │   └── proposal.md
  ├── itsm-advanced-nodes/        ⑤ 脚本/边界/子流程 (依赖
  ②③)
  │   ├── .openspec.yaml
  │   └── proposal.md
  ├── itsm-bpmn-designer/         ⑥ 可视化设计器 (依赖
  ①②③④⑤)
  │   ├── .openspec.yaml
  │   └── proposal.md
  └── itsm-runtime-tracking/      ⑦ 运行时追踪 (依赖 ⑥)
      ├── .openspec.yaml
      └── proposal.md
