# Support Bundle

Support Bundle Preview 用于列出诊断包可能包含的 metadata、hash、版本、计数和路径引用。

当前 preview 不创建压缩包、不复制项目文件、不上传数据，也不读取 secret value。默认排除：

- secret 和 credential。
- 用户文件原文。
- 未授权的项目源码或 workflow execution 内容。
- artifact store 中未显式允许导出的内容。

真实 export、上传或远程支持流程尚未开放，需要单独的 redaction、approval、audit 和目标授权。
