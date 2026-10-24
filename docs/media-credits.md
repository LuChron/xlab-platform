# 媒体与品牌素材说明

机器可读的逐文件记录位于
[frontend/public/media/credits.json](../frontend/public/media/credits.json)。本页说明公开仓库中的使用边界。

## 浙江大学官网与 X-Lab 素材

文件名以 xlab-official- 开头的照片来自浙江大学官网相关报道，摄影署名为 LI Jiaxuan；
X-Lab 标志来自 X-Lab 官网。来源页面和原始资源地址已记录在 credits.json 中。

来源页面未标示 Creative Commons 等开放许可，因此这些素材的原始版权仍归原权利人所有。
当前仓库仅将其用于课程项目展示与界面原型。若要用于商业产品、公开宣传或长期线上服务，
应先取得权利人许可；无法确认时可替换为团队自摄照片或明确开放授权的素材。

## Wikimedia Commons 素材

其余部分照片与视频来自 Wikimedia Commons，并分别采用 CC BY、CC BY-SA 或公有领域许可。
作者、作品名、许可链接及是否要求署名均记录在 credits.json 中。再分发或修改时应继续满足
对应许可的署名与相同方式共享要求。

## 维护要求

- 新增媒体时同步补充 credits.json，不把来源说明散落在页面正文。
- 替换或裁切 CC BY-SA 图片时检查相同方式共享要求。
- 发布 Release 或部署线上版本前，再核对实际打包进 frontend/public/media 的文件。
- 代码许可证不自动覆盖图片、视频、Logo 和其他第三方素材。
