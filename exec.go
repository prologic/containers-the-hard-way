package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func getPidForRunningContainer(containerID string) int {
	containers, err := getRunningContainers()
	if err != nil {
		log.Fatalf("Unable to get running containers: %v\n", err)
	}
	for _, container := range containers {
		if container.containerId == containerID {
			return container.pid
		}
	}
	return 0
}

func execInContainer(containerId string) {
	pid := getPidForRunningContainer(containerId)
	if pid == 0 {
		log.Fatalf("No such container!")
	}
	baseNsPath := "/proc/" + strconv.Itoa(pid) + "/ns"
	ipcFd, ipcErr := os.Open(baseNsPath + "/ipc")
	mntFd, mntErr := os.Open(baseNsPath + "/mnt")
	netFd, netErr := os.Open(baseNsPath + "/net")
	pidFd, pidErr := os.Open(baseNsPath + "/pid")
	utsFd, utsErr := os.Open(baseNsPath + "/uts")

	if ipcErr != nil || mntErr != nil || netErr != nil ||
		pidErr != nil || utsErr != nil {
		log.Fatalf("Unable to open namespace files!")
	}

	unix.Setns(int(ipcFd.Fd()), syscall.CLONE_NEWIPC)
	unix.Setns(int(mntFd.Fd()), syscall.CLONE_NEWNS)
	unix.Setns(int(netFd.Fd()), syscall.CLONE_NEWNET)
	unix.Setns(int(pidFd.Fd()), syscall.CLONE_NEWPID)
	unix.Setns(int(utsFd.Fd()), syscall.CLONE_NEWUTS)

	containerConfig, err := getRunningContainerInfoForId(containerId)
	if err != nil {
		log.Fatalf("Unable to get container configuration")
	}
	imageNameAndTag := strings.Split(containerConfig.image, ":")
	exists, imageShaHex := imageExistByTag(imageNameAndTag[0], imageNameAndTag[1])
	if !exists {
		log.Fatalf("Unable to get image details")
	}
	containerMntPath := getGockerContainersPath() + "/" + containerId + "/fs/mnt"
	createCGroups(containerId, false)

	args := []string{"exec-child"}
	args = append(args, "--img="+imageShaHex)
	args = append(args, "--root="+containerMntPath)
	args = append(args, os.Args[3:]...)
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	doOrDie(cmd.Run())
}
