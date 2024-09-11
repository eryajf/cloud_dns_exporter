# 贡献者指南

欢迎反馈、bug 报告和拉取请求，可点击[issue](https://github.com/eryajf/cloud_dns_exporter/issues) 提交.

如果你是第一次进行 GitHub 协作，可参阅： [协同开发流程](https://howtosos.eryajf.net/HowToStartOpenSource/01-basic-content/03-collaborative-development-process.html)

1. 项目使用`golangci-lint`进行检测，提交 pr 之前请在本地执行 `make lint` 并通过。

2. 如有新的服务商，则需要注意在对应readme也做好对应标注说明。

3. 如果domain_id或record_id为空，则可通过调用 public.GetID() 进行填充。