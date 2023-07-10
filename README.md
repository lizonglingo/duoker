# duoker
实现一个简易的类似 docker 的程序，理解容器技术原理。

代码参照 [bilibili蓝老师的视频](https://space.bilibili.com/274721678/channel/collectiondetail?sid=70487)

# git 使用

使用 v2ray 配置 git 加速：

```shell
root@nm:/work-place# git config --global http.https://github.com.proxy https://10.0.21.66:10809
root@nm:/work-place# git config --global https.https://github.com.proxy https://10.0.21.66:10809
root@nm:/work-place# git config --global http.https://github.com.proxy socks5://10.0.21.66:10808
root@nm:/work-place# git config --global https.https://github.com.proxy socks5://10.0.21.66:10808
```

列出远程分支：`git branch -r`

拉取并切换本地不存在，但是远程存在的分支：`git checkout -b 本地分支名 origin/远程分支名`。
例如远程有一个 `dev` 分支，本地没有，就用 `git checkout -b dev origin/dev` 拉取并切换到这个分支。

将当前路径下修改的代码，跟踪新增加的文件：

```shell
git add .
```

查看目前跟踪文件的状态：

```shell
git status
```

有的文件想忽略而不提交到仓库，就在 `.git` 同目录下添加 `.gitignore` 记录
需要被忽略的文件。

文件 `.gitignore` 的格式规范如下：

- 所有空行或者以 `#` 开头的行都会被 Git 忽略。

- 可以使用标准的 glob 模式匹配，它会递归地应用在整个工作区中。

- 匹配模式可以以 `/` 开头防止递归。

- 匹配模式可以以 `/` 结尾指定目录。

- 要忽略指定模式以外的文件或目录，可以在模式前加上叹号 `!` 取反。

下面是一个例子：

```gitignore
# 忽略所有的 .a 文件
*.a

# 但跟踪所有的 lib.a，即便你在前面忽略了 .a 文件
!lib.a

# 只忽略当前目录下的 TODO 文件，而不忽略 subdir/TODO
/TODO

# 忽略任何目录下名为 build 的文件夹
build/

# 忽略 doc/notes.txt，但不忽略 doc/server/arch.txt
doc/*.txt

# 忽略 doc/ 目录及其所有子目录下的 .pdf 文件
doc/**/*.pdf
```
将代码提交到本地，通过 `-m` 添加本次提交的信息。

```shell
git commit -m "Story 182: Fix benchmarks for speed"
```

查看提交历史。

```shell
git log
```

将本地提交推送到远程仓库的某个分支。

```shell
git push origin dev
```

# 容器命名空间

纸上得来终觉浅，我们都能说出容器的底层技术是依赖 namespace，cgroup，union fs 等等，如果不动手用用 Linux 的这些能力，感觉还是
一知半解的。

## Namespace

Namespace 主要用于进程隔离，所隔离的信息包括主机名、进程Pid空间、mount命名空间、网络等。

这里着重实践一下 mount [命名空间的隔离](https://mp.weixin.qq.com/s?__biz=MzU3NjY5MjY2Ng==&mid=2247486468&idx=1&sn=5e2e537115146348d7bc137ddf4a31ff&chksm=fd1144d1ca66cdc771294afd20f6324c9c49214042362aa622e5809c0c8f9f834aeca4a50bcb&scene=178&cur_album_id=2810774774825123840#rd)




