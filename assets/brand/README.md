# AreaFlow Brand Assets

`final/` 是 AreaFlow 品牌素材包的唯一权威交付目录，`brand-manifest.json` 是机器可读规格源。

## 闭环结构

```text
品牌几何与字标代码
  -> canonical SVG
  -> brand-manifest.json
  -> PNG / ICO / ICNS / PDF / TIFF / 原生平台包
  -> brand:validate
  -> Makefile 与 CI 门禁
```

## 命令

```bash
npm ci
npm run brand:export
npm run brand:validate
```

`brand:export` 默认补齐缺失素材，`npm run brand:export -- --refresh` 全量重建；`brand:validate` 检查清单身份、SVG、尺寸、透明度、
ICO/ICNS、iOS slots、Android adaptive icon、CMYK TIFF 的 300 DPI 元数据、文档和目录卫生。

正式使用规则、颜色、最小尺寸和文件选择见 [`final/README.md`](final/README.md)。
