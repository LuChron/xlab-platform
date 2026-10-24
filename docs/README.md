# 文档目录

运行方式和当前功能以根目录 [README.md](../README.md) 为准。本目录保存产品设计、实现记录、素材说明和项目补充材料。

## 产品与实现

- [product-design.md](product-design.md)：业务边界、信息架构、状态机与链上链下划分。
- [implementation-roadmap.md](implementation-roadmap.md)：实施顺序与完成情况。
- [design-options/](design-options/)：早期视觉方向对比，作为设计过程记录。
- [screenshots/](screenshots/)：README 使用的当前页面截图。

## 发布与合规

- [GitHub发布清单.md](GitHub发布清单.md)：仓库命名、推送步骤与公开前检查。
- [media-credits.md](media-credits.md)：图片、视频和品牌素材的授权边界。
- [SECURITY.md](../SECURITY.md)：演示鉴权、网络暴露和生产化注意事项。

## 补充材料

- [项目介绍.md](项目介绍.md)：背景、架构、链上设计与功能说明。
- [答辩PPT.md](答辩PPT.md)：页面讲解顺序与常见问题。
- [组员分工.md](组员分工.md)：团队成员与贡献记录模板。
- [演示视频素材/](演示视频素材/)：完整流程视频与关键步骤静帧。

## 文档维护规则

- 功能、示例数据和启动命令变化时，同时更新根 README 和项目介绍。
- 页面结构变化后重新生成 screenshots；旧演示视频保留录制日期，避免与当前界面混淆。
- Fabric 证书、私钥、.env、构建产物与本地缓存不得进入仓库。
- 媒体文件的来源信息写入 frontend/public/media/credits.json，不要只写在页面正文里。
