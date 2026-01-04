# 前端配置推荐方案

前端现在通过 `src/config/index.ts` 统一读取配置。默认值写在代码里，同时自动合并：

1. `NEXT_PUBLIC_*` 系列环境变量（构建时确定）
2. 运行期注入的 `window.__ROBUSTA_RUNTIME_CONFIG__`

这样可以在容器内通过挂载静态文件而无需重新构建即可调整地址。

## 运行时配置示例

1. 拷贝 `config/runtime.sample.json` 到 `public/config/runtime.json` 并改成目标环境的地址。
2. 在 `app/layout.tsx` 或入口 HTML 中注入脚本，将 JSON 加载到全局变量：

```html
<script>
  fetch('/config/runtime.json')
    .then(r => r.json())
    .then(cfg => { window.__ROBUSTA_RUNTIME_CONFIG__ = cfg })
    .catch(() => {})
</script>
```

`appConfig` 会自动合并此配置，因此组件中只需要：

```ts
import { appConfig } from '@/config'

appConfig.apiBaseUrl // => 最终可用的 API 地址
```

如果某些页面需要临时覆盖，可以使用 `resolveConfig(overrides)`。
