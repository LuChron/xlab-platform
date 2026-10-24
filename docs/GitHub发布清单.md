# GitHub 发布清单

## 推荐仓库信息

- 仓库名：xlab-resource-chain
- 展示名称：X-Lab 创新项目入驻与设备协作平台
- 简介：A Hyperledger Fabric platform for X-Lab project residency, space allocation and equipment collaboration.
- Topics：hyperledger-fabric、blockchain、vue3、golang、gin、vite、smart-contract、course-project
- 默认分支：main

xlab-resource-chain 比 xlab-platform 更能说明项目的核心：X-Lab、实体资源协作和链上留痕；
同时没有把实现限定成某个页面或单一设备借用系统。

## 推送前

- 运行 ./deploy/check.sh，确保后端、链码、前端和部署脚本全部通过。
- 运行 git status，确认没有 .env、证书、私钥、日志、node_modules、dist 或本地缓存。
- 填写 docs/组员分工.md 中的真实姓名与贡献值；公开仓库不需要学号等敏感信息。
- 确认 README 截图和演示视频是否与最终界面一致。
- 阅读 docs/media-credits.md，确认公开发布所使用图片的许可范围。
- 决定代码许可证。未添加 LICENSE 时，GitHub 默认保留全部权利。

## 当前仓库体积

当前 Git 历史约 145 MiB，最大单文件是约 38 MiB 的演示视频，低于 GitHub 单文件
100 MiB 限制，可以直接推送，但首次上传和克隆会较慢。若希望公开仓库更轻，可在
正式推送前另建一个只包含当前版本的干净仓库，或把演示视频放到 GitHub Release。
不要在已有协作仓库中直接重写历史。

## 首次推送

在 GitHub 创建空仓库后执行：

    git branch -M main
    git remote add origin git@github.com:<your-account>/xlab-resource-chain.git
    git push -u origin main

如果使用 HTTPS，把远程地址替换为：

    https://github.com/<your-account>/xlab-resource-chain.git

## GitHub 首页建议

- About 中填写上面的英文简介和 Topics。
- Social preview 可使用 docs/screenshots/home.jpg。
- README 顶部保持中文项目名，便于课程展示；英文简介放在仓库 About 即可。
- 暂不把演示账号用于公网长期部署；公开在线演示前先按 SECURITY.md 完成鉴权加固。
