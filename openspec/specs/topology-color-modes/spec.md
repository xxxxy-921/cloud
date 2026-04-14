# Capability: topology-color-modes

## Purpose
定义 APM 拓扑图的多种着色模式（Error Rate、Latency P95、Throughput），包括切换器、阈值映射和图例。

## Requirements

### Requirement: Color-by mode switcher
拓扑页面 SHALL 提供 Color by 下拉切换器，支持三种着色模式：Error Rate（默认）、Latency P95、Throughput。

#### Scenario: Default mode
- **WHEN** 页面加载
- **THEN** 默认使用 Error Rate 着色模式

#### Scenario: Switch to Latency mode
- **WHEN** 用户选择 "Latency P95"
- **THEN** 所有节点环和边颜色根据 P95 延迟重新映射（低=绿、中=黄、高=红）

#### Scenario: Switch to Throughput mode
- **WHEN** 用户选择 "Throughput"
- **THEN** 所有节点环颜色根据 requestCount 映射（渐变色阶，高流量=深色）

### Requirement: Color mapping thresholds
各着色模式 SHALL 有合理的阈值分级。

#### Scenario: Error Rate thresholds
- **WHEN** Color by = Error Rate
- **THEN** <1% = 绿色，1-5% = 黄色，>5% = 红色

#### Scenario: Latency thresholds
- **WHEN** Color by = Latency P95
- **THEN** 基于当前数据集的 P50/P90 动态计算阈值（低于 P50 = 绿，P50-P90 = 黄，高于 P90 = 红）

### Requirement: Legend updates with color mode
底部图例 SHALL 随 color-by 模式切换更新标签和色阶描述。

#### Scenario: Legend for error rate mode
- **WHEN** Color by = Error Rate
- **THEN** 图例显示 "Healthy (<1%)" / "Warning (1-5%)" / "Critical (>5%)"

#### Scenario: Legend for latency mode
- **WHEN** Color by = Latency P95
- **THEN** 图例显示 "Low" / "Medium" / "High" 及对应色阶
