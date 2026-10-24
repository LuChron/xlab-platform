# 文档导航

本目录同时服务于课程提交、项目维护和 GitHub 展示。运行命令与最新功能以根目录
[README.md](../README.md) 为准。

## 课程交付

- [项目介绍.md](项目介绍.md)：背景、架构、链上设计、功能与工程规模。
- [答辩PPT.md](答辩PPT.md)：7 分钟演示脚本与常见问题。
- [组员分工.md](组员分工.md)：待小组填写姓名与贡献值的提交模板。
- [演示视频素材/](演示视频素材/)：全流程视频、关键步骤静帧与版本说明。

## 设计与实现

- [product-design.md](product-design.md)：业务边界、信息架构、状态机与链上链下划分。
- [implementation-roadmap.md](implementation-roadmap.md)：实施顺序与完成情况。
- [design-options/](design-options/)：早期视觉方向对比，作为设计过程记录。
- [screenshots/](screenshots/)：README 使用的当前页面截图。

## 发布与合规

- [GitHub发布清单.md](GitHub发布清单.md)：仓库命名、推送步骤与公开前检查。
- [media-credits.md](media-credits.md)：图片、视频和品牌素材的授权边界。
- [SECURITY.md](../SECURITY.md)：演示鉴权、网络暴露和生产化注意事项。

## 文档维护规则

- 功能数量、演示数据和启动命令变化时，同时更新根 README、项目介绍和答辩脚本。
- 页面结构变化后重新生成 screenshots；旧演示视频保留录制日期，避免与当前界面混淆。
- Fabric 证书、私钥、.env、构建产物与本地缓存不得进入仓库。
- 媒体文件的来源信息写入 frontend/public/media/credits.json，不要只写在页面正文里。
