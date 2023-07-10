// Package workspace 用于为容器进行根目录、工作目录的挂载.
package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const (
	// 这里为了目录清晰 都用绝对路径
	mntPath        = "/workplace/duoker/rootfs/mnt"
	workLayerPath  = "/workplace/duoker/rootfs/work"
	writeLayerPath = "/workplace/duoker/rootfs/wlayer"
	// 我们的 base 文件目录就在编译好的 duoker 的目录下 所以这个用相对目录
	imagePath = "ubuntu-base-22.04-base-amd64"
	// put_old 为 new_root 的子文件夹 所以也用相对目录
	mntOldPath = ".old"
)

// workerLayer 生成容器中的工作目录.
func workerLayer(containerName string) string {
	return fmt.Sprintf("%s/%s", workLayerPath, containerName)
}

// mntLayer 挂载目录.
func mntLayer(containerName string) string {
	return fmt.Sprintf("%s/%s", mntPath, containerName)
}

// writeLayer 读写层
func writeLayer(containerName string) string {
	return fmt.Sprintf("%s/%s", writeLayerPath, containerName)
}

// mntOldLayer put_old 目录
// 用于 pivot_root.
func mntOldLayer(containerName string) string {
	return fmt.Sprintf("%s/%s", mntLayer(containerName), mntOldPath)
}

// SetMountNamespace 为容器设置挂载命名空间.
// 1. 创建 overlay 联合文件系统
//		1.1 配置只读层  也就是容器内的根文件系统
//		1.2 配置 work 空间 容器的工作目录
//		1.3 配置 write 作为容器的读写层
//		1.4 进行挂载
func SetMountNamespace(containerName string) error {
	// 配置挂载目录
	if err := os.Mkdir(mntLayer(containerName), 0700); err != nil {
		return fmt.Errorf("mkdir mntlayer fail err=%s", err)
	}

	// 配置容器内工作目录
	if err := os.Mkdir(workerLayer(containerName), 0700); err != nil {
		return fmt.Errorf("mkdir work layer fail err=%s", err)
	}

	// 配置读写层目录
	if err := os.Mkdir(writeLayer(containerName), 0700); err != nil {
		return fmt.Errorf("mkdir write layer fail err=%s", err)
	}

	// 1. 目录创建好后 进行 overlay 的挂载
	// 	  这里会把我们的 Ubuntu base 目录挂载到 mntlayer 所在的文件夹下
	if err := syscall.Mount("overlay", mntLayer(containerName), "overlay", 0,
		fmt.Sprintf("upperdir=%s,lowerdir=%s,workdir=%s",
			writeLayer(containerName), imagePath, workerLayer(containerName)),
	); err != nil {
		return fmt.Errorf("mount overlay fail err=%s", err)
	}

	// 2. 抽离上一版本 main.go 切换根目录的代码放在这里
	// 	  systemd 启动默认是 share 模式 这样就不能隔离挂载可见性 所以当前根目录下所有目录设为 private 模式
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("reclare rootfs private fail err=%s", err)
	}

	// 3. 挂载新的根目录 也就是我们之前创建的 mnt
	//    后面 pivot_root 会使用 作为 new_root
	//    在 main 的代码中 我们已经重新设定了 mnt 命名空间了
	//    bind 相当于一个硬链接 无论在哪一方读写 都会反映到另一方
	if err := syscall.Mount(
		mntLayer(containerName), mntLayer(containerName), "bind", syscall.MS_BIND|syscall.MS_REC, "",
	); err != nil {
		return fmt.Errorf("mount rootfs in new mnt space fail err=%s", err)
	}

	// 4. 配置 pivot_root 的 put_old 目录
	if err := os.Mkdir(mntOldLayer(containerName), 0700); err != nil {
		return fmt.Errorf("mkdir .old for pivot_root fail err=%s", err)
	}

	// 5. 执行 pivot_root
	if err := syscall.PivotRoot(mntLayer(containerName), mntOldLayer(containerName)); err != nil {
		return fmt.Errorf("pivot root  fail err=%s", err)
	}

	return nil
}

// DelMntNamespace 清理 overlay 文件系统和涉及到的的文件夹.
func DelMntNamespace(containerName string) error {
	if err := unmountAndDelPath(mntLayer(containerName)); err != nil {
		return err
	}
	if err := unmountAndDelPath(workerLayer(containerName)); err != nil {
		return err
	}
	if err := unmountAndDelPath(writeLayer(containerName)); err != nil {
		return err
	}
	return nil
}

// unmountAndDelPath 在容器运行结束后 卸载挂载点并删除对应的目录.
func unmountAndDelPath(path string) error {
	_, err := exec.Command("umount", path).CombinedOutput()
	if err != nil {
		// return fmt.Errorf("umount fail path=%s err=%s", path, err)
		fmt.Errorf("umount fail path=%s err=%s", path, err)
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove dir fail path=%s err=%s", path, err)
	}
	return nil
}
