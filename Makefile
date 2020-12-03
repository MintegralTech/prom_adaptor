# 获取源码最近一次 git commit log，包含 commit sha 值，以及 commit message
GitCommitLog=$(shell git log --pretty=oneline -n 1)
# 检查源码在git commit 基础上，是否有本地修改，且未提交的内容
#GitStatus=`git status -s`
#echo ${GitStatus}
# 获取当前时间
BuildTime=$(shell date +'%Y.%m.%d.%H%M%S')
#git tag
GitTag=$(shell git describe --tags)
# 获取 Go 的版本
BuildGoVersion=$(shell go version)

# 将以上变量序列化至 LDFlags 变量中
LDFlags="\
    -X 'main.GitCommitLog=${GitCommitLog}' \
    -X 'main.BuildTime=${BuildTime}' \
    -X 'main.GitTag=${GitTag}' \
    -X 'main.BuildGoVersion=${BuildGoVersion}' \
"
BinName="server"
ImageTag="unknown"
all: dbuild build

.PHONY: dbuild
dbuild:
	@echo "start dbuild, $(LDFlags)"
	docker build --build-arg LDFlags=$(LDFlags) -t  michaeldiao1/aggregator:$(ImageTag) .
	docker push michaeldiao1/aggregator:$(ImageTag)
	@echo "dbuild done."

.PHONY: build
build:
	@echo "start build"
	go build  -ldflags $(LDFlags) -o $(BinName) ./cmd/server.go
	@echo "build done."

clean:
	rm -rf $(BinName)