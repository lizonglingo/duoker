package main

import (
	"duoker/config"
	"duoker/log"
	"duoker/network"
	"duoker/workspace"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// ./duoker run containerName /bin/sh

func main() {
	switch os.Args[1] {
	case "run":
		// 首先进行网络初始化
		//		1. 在宿主机上创建网桥
		//		2. 为网桥配置基础信息 如 网段 子网地址等
		// 		3. 为宿主机配置内网的 NAT
		if err := network.Init(); err != nil {
			log.Error("net work fail err=%s", err)
			return
		}
		fmt.Println(config.Banner())
		// 在一个新的命名空间
		// 打印本进程和父进程的 Pid
		fmt.Println("run pid ", os.Getpid(), "ppid", os.Getppid())
		// 这里拿到的 initCmd 就是 duoker 进程连接
		// 在后面还要执行一次我们编译好的这个 duoker 程序
		initCmd, err := os.Readlink("/proc/self/exe")
		if err != nil {
			log.Error("get init process error %s", err)
			return
		}
		os.Args[1] = "init"
		cmd := exec.Command(initCmd, os.Args[1:]...)
		// 启动一个新的命名空间 并进行配置
		// syscall.CLONE_NEWUTS	对主机名进行隔离
		// syscall.CLONE_NEWPID	对pid空间进行隔离
		// syscall.CLONE_NEWNS	对mount命名空间进行隔离
		// syscall.CLONE_NEWNET	对网络进行隔离
		// syscall.CLONE_NEWIPC	对进程通信组件进行隔离（消息队列）
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
				syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
		}
		// 获取当前的环境变量
		// 配置标准输入输出 1
		cmd.Env = os.Environ()
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// cmd.Run()	会等待命令结束
		// cmd.Start()	不会等待命令结束
		// 从上个版本的 cmd.Run() 变为 cmd.Start()
		err = cmd.Start()
		if err != nil {
			fmt.Println(err)
		}
		// 等待子进程完全启动
		time.Sleep(2 * time.Second)

		// 创建 Veth Peer 连接到容器和宿主机的 Bridge
		if err := network.ConfigDefaultNetworkInNewNet(cmd.Process.Pid); err != nil {
			log.Error("config network fail %s", err)
		}

		// 在这里等待子进程的结束 因为前面使用的 cmd.Start 执行的命令
		cmd.Wait()
		workspace.DelMntNamespace(os.Args[2])
		return
	case "init":
		var (
			containerName = os.Args[2]
			cmd           = os.Args[3]
		)
		log.Info("Wait  SIGUSR2 signal arrived ....")
		// 等待父进程网络命名空间设置完毕
		// 这里 WaitParentSetNewNet() 中使用 channel 阻塞了一个信号 等待父进程设置完之后再通知
		network.WaitParentSetNewNet()
		if err := workspace.SetMountNamespace(containerName); err != nil {
			log.Error("SetMntNamespace %s", err)
			return
		}
		syscall.Chdir("/")
		defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
		syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
		err := syscall.Exec(cmd, os.Args[3:], os.Environ())
		if err != nil {
			log.Error("exec proc fail %s", err)
			return
		}
		log.Error("forever not  exec it ")
		return
	default:
		log.Error("not valid cmd")
	}
}

//
//func main() {
//
//	switch os.Args[1] {
//	case "run":
//		// 打印本进程的 Pid
//		fmt.Println("run pid ", os.Getpid(), "ppid", os.Getppid())
//		// 这里拿到的 initCmd 就是 duoker 进程连接
//		// 在后面还要执行一次我们编译好的这个 duoker 程序
//		initCmd, err := os.Readlink("/proc/self/exe")
//		fmt.Println("initCmd symbolic link: ", initCmd)
//		if err != nil {
//			fmt.Println("get init process error ", err)
//			return
//		}
//
//		// 我们需要获取容器名 根据容器名（唯一） 去创建其对应的 overlay 联合文件系统
//		// 保证镜像只读层的特性
//		containerName := os.Args[2]
//
//		// 重写第一个参数 run -> init 可以执行到 case "init" 中
//		os.Args[1] = "init"
//		cmd := exec.Command(initCmd, os.Args[1:]...)
// 启动一个新的命名空间 并进行配置
// syscall.CLONE_NEWUTS	对主机名进行隔离
// syscall.CLONE_NEWPID	对pid空间进行隔离
// syscall.CLONE_NEWNS	对mount命名空间进行隔离
// syscall.CLONE_NEWNET	对网络进行隔离
// syscall.CLONE_NEWIPC	对进程通信组件进行隔离（消息队列）
//		cmd.SysProcAttr = &syscall.SysProcAttr{
//			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
//				syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
//		}
//		cmd.Env = os.Environ() // 获取当前的环境变量
//		// 设置标准输入输出
//		cmd.Stdin = os.Stdin
//		cmd.Stdout = os.Stdout
//		cmd.Stderr = os.Stderr
//
//		// cmd.Run()	会等待命令结束
//		// cmd.Start()	不会等待命令结束
//		err = cmd.Run()
//		if err != nil {
//			fmt.Println("cmd run error:", err)
//		}
//
//		// 运行结束后清理挂载命名空间 并删除对应文件夹
//		err = workspace.DelMntNamespace(containerName)
//		if err != nil {
//			fmt.Println("clean overlayfs and related dir error:", err)
//		}
//		fmt.Println("Bye!")
//		return
//	case "init":
//		var (
//			containerName = os.Args[2]
//			cmd           = os.Args[3]
//		)
//		// 为容器配置 overlayfs 和相关的目录
//		if err := workspace.SetMountNamespace(containerName); err != nil {
//			fmt.Println(err)
//			return
//		}
//		// 执行 pivot_root 需要手动切到新的根路径
//		syscall.Chdir("/")
//		// 挂载 /proc 使其可以获得 pid 信息
//		defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
//		syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
//		err := syscall.Exec(cmd, os.Args[3:], os.Environ())
//		if err != nil {
//			fmt.Println("exec proc fail ", err)
//			return
//		}
//		fmt.Println("forever exec it ")
//		return
//	default:
//		fmt.Println("not valid cmd")
//	}
//}
