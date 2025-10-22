# 前端原型区

此目录存放尚未完全融入正式前端应用的原型实现，便于演示或后续迭代。  
所有共享组件与类型都放置在 `@prototypes` 命名空间下，避免与生产代码冲突。  
引用方式示例：

```ts
import { AlertList } from '@prototypes/alerts/components/alerts/AlertList'
```

在将原型提升为正式功能前，需把相关代码迁移到 `src/app`、`src/components` 等正式路径，并补充测试与文档。
