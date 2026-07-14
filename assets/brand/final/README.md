# AreaFlow Brand Assets

本目录保存 AreaFlow 当前品牌素材包。机器可读规格位于上一级 `brand-manifest.json`，生成、验证与持续门禁见上一级 `README.md`。

## 目录

- `areaflow-app-icon-dark.svg` / `areaflow-app-icon-light.svg`：完整 App/PWA 图标源。
- `areaflow-app-icon-small-dark.svg` / `areaflow-app-icon-small-light.svg`：16px、32px 和 48px 小尺寸简化源。
- `areaflow-app-icon-opaque-*.svg`：Apple Touch Icon 和 App Store 使用的不透明全出血源。
- `areaflow-app-icon-maskable-*.svg`：PWA maskable 全出血源，主体位于安全区内。
- `areaflow-logo-mark-*.svg`：独立标志源；`mono` 文件为单色版本。
- `areaflow-logo-symbol-*.svg`：无底板透明 Symbol，针对目标背景适配对比。
- `areaflow-logo-lockup*.svg`：横向 Logo，包含默认、深浅背景、轮廓化和单色版本。
- `areaflow-wordmark-*.svg`：不带图标的纯字标源。
- `areaflow-logo-stacked-*.svg`：竖向堆叠 Logo 源。
- `app-icon/`：常规 App icon PNG 为 `16/32/48/64/128/180/192/256/512/1024` 深浅两套，并包含 opaque 与 maskable 导出。
- `mark/`、`symbol/`、`lockup/`、`wordmark/`、`stacked/`：常用 PNG 导出。
- `favicon/`：`16/32/48` PNG 与多尺寸 ICO。
- `social/`：`1200x630` 深浅社交预览图；无后缀文件为浅色兼容入口。
- `native/macos/`：`AreaFlow.icns` 与完整 `.iconset`。
- `native/ios/`：iPhone、iPad 和 App Store marketing 尺寸的 `AreaFlowAppIcon.appiconset`。
- `native/android/res/`：Android adaptive icon 前景、背景色和 v26 XML。
- `native/windows/AreaFlow.ico`：包含 `16/24/32/48/64/128/256` 的 Windows 应用图标。
- `print/`：浅色/深色背景的 outlined SVG、矢量 PDF 与 300 DPI CMYK TIFF。
- `areaflow-brand-overview.png`：完整数字品牌素材总览。

## 使用边界

- 主体结构固定为调度盘、双 Flow 轨迹、执行节点、完成节点和弱底线，不再改变核心构图。
- 深浅版本保持同一结构，只做背景、灰度和对比适配。
- 横向字标中 `Area` 使用中性色，`Flow` 使用青绿、青色、琥珀和珊瑚红渐变。
- 16px、32px 和 48px 必须使用 small 源，避免调度刻度与中心细节糊成一团。
- 常规 App icon 保留圆角透明边；opaque 与 maskable 必须全画布不透明。
- `lockup/wordmark/stacked/symbol` 的 `dark/light` 表示目标背景；`mono` 的 `dark/light` 表示墨线明暗。
- 对外直接分发横向 SVG 时优先使用 `outlined` 版本，避免字体替换。
- 社交预览图定位为“AI 开发执行治理平台”，主标题为“把需求编排成可审计的软件交付”。
- 横向 Logo 最小屏幕宽度为 120px，印刷最小宽度为 25mm；低于 48px 使用 small icon，最低不得小于 16px。
- Apple 与 Windows 原生包使用 opaque dark 源；Android 使用透明 Symbol 和品牌深色背景。
- 印刷 CMYK 文件是从 sRGB 品牌色换算的通用起点，正式生产仍需印厂打样。

## 标准色

| 名称 | HEX | 用途 |
|---|---|---|
| Flow Ink 950 | `#07191D` | 深色背景、App icon 底板 |
| Flow Ink 900 | `#09272D` | 浅色背景字标、描边 |
| Scheduler Mint | `#36D9A6` | 起始节点与 Flow 轨迹 |
| Control Cyan | `#18BFC7` | 调度核心与主强调 |
| Evidence Amber | `#F5B02E` | 执行证据节点 |
| Completion Coral | `#F46D5E` | 完成节点与风险强调 |
| Control Mist | `#F4FBF8` | 深色背景文字 |
| Surface Mist | `#F1FAF7` | 浅色图标底板 |

## 生成与校验

```bash
npm ci
npm run brand:export
npm run brand:validate
```

生成器读取 `brand-manifest.json` 的尺寸、透明度、平台来源和印刷 DPI；校验器覆盖全部清单输出、原生包、印刷包和目录卫生。

默认导出只补齐缺失文件；需要从当前 SVG 全量重建时运行 `npm run brand:export -- --refresh`。
