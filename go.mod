module github.com/k8shop/tun2socks

go 1.25.3

require gvisor.dev/gvisor v0.0.0-20251114082540-5928db6b1cc3
require github.com/k8shop/water v1.0.0

// gvisor默认下载的版本是master分支，
// 该分支有包重复定义报错问题 (Found several packages in xxx)

// 解决步骤:
// 1、添加 replace gvisor.dev/gvisor => github.com/google/gvisor go
//  然后执行 go mod tidy 得到go分支对应的 time+hash正确版本号
// 2、将该版本号替换到 require指令后面(推荐)
//  或者把replace指令添加在外层 (注意被依赖包中的replace指令是不生效的)


// replace指令:
// 1、pkg.go.dev包替换为 github源码、本地目录
// 2、只能用在项目当前目录go.mod (在依赖包中被忽略)

require (
	github.com/google/btree v1.1.2 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/time v0.12.0 // indirect
)
