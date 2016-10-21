package sys

import (
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/quantum/castle/pkg/util/exec"
)

// request the current user once and stash it in this global variable
var currentUser *user.User

func GetDeviceFilesystem(device string, executor exec.Executor) (string, error) {
	cmd := fmt.Sprintf("get filesystem type for %s", device)
	devFS, err := executor.ExecuteCommandPipeline(
		cmd,
		fmt.Sprintf(`df --output=source,fstype | grep '^/dev/%s ' | awk '{print $2}'`, device))
	if err != nil {
		return "", fmt.Errorf("command %s failed: %+v", cmd, err)
	}

	return devFS, nil
}

func FormatDevice(devicePath string, executor exec.Executor) error {
	cmd := fmt.Sprintf("mkfs.ext4 %s", devicePath)
	if err := executor.ExecuteCommand(cmd, "sudo", "mkfs.ext4", devicePath); err != nil {
		return fmt.Errorf("command %s failed: %+v", cmd, err)
	}

	return nil
}

// look up the UUID for a disk.
func GetDiskUUID(deviceName string, executor exec.Executor) (string, error) {
	cmd := fmt.Sprintf("get disk %s uuid", deviceName)
	uuid, err := executor.ExecuteCommandPipeline(
		cmd,
		fmt.Sprintf(`sgdisk -p /dev/%s | grep 'Disk identifier (GUID)' | awk '{print $4}'`, deviceName))
	if err != nil {
		log.Printf("unknown disk uuid for /dev/%s", deviceName)
		return "", nil
	}

	return uuid, nil
}

// look up the mount point of the given device.  empty string returned if device is not mounted.
func GetDeviceMountPoint(deviceName string, executor exec.Executor) (string, error) {
	cmd := fmt.Sprintf("get mount point for %s", deviceName)
	mountPoint, err := executor.ExecuteCommandPipeline(
		cmd,
		fmt.Sprintf(`mount | grep '^/dev/%s on' | awk '{print $3}'`, deviceName))
	if err != nil {
		return "", fmt.Errorf("command %s failed: %+v", cmd, err)
	}

	return mountPoint, nil
}

func GetDeviceFromMountPoint(mountPoint string, executor exec.Executor) (string, error) {
	cmd := fmt.Sprintf("get device from mount point %s", mountPoint)
	device, err := executor.ExecuteCommandPipeline(
		cmd,
		fmt.Sprintf(`mount | grep 'on %s ' | awk '{print $1}'`, mountPoint))
	if err != nil {
		return "", fmt.Errorf("command %s failed: %+v", cmd, err)
	}

	return device, nil
}

func MountDevice(devicePath, mountPath string, executor exec.Executor) error {
	return MountDeviceWithOptions(devicePath, mountPath, "", executor)
}

// comma-separated list of mount options passed directly to mount command
func MountDeviceWithOptions(devicePath, mountPath, options string, executor exec.Executor) error {
	var args []string
	if options != "" {
		args = []string{"mount", "-o", options, devicePath, mountPath}
	} else {
		args = []string{"mount", devicePath, mountPath}
	}

	os.MkdirAll(mountPath, 0755)
	cmd := fmt.Sprintf("mount %s", devicePath)
	if err := executor.ExecuteCommand(cmd, "sudo", args...); err != nil {
		return fmt.Errorf("command %s failed: %+v", cmd, err)
	}

	return nil
}

func UnmountDevice(devicePath string, executor exec.Executor) error {
	cmd := fmt.Sprintf("umount %s", devicePath)
	if err := executor.ExecuteCommand(cmd, "sudo", "umount", devicePath); err != nil {
		cmdErr, ok := err.(*exec.CommandError)
		if ok && cmdErr.ExitStatus() == 32 {
			log.Printf("ignoring exit status 32 from unmount of device %s, err:%+v", devicePath, cmdErr)
		} else {
			return fmt.Errorf("command %s failed: %+v", cmd, err)
		}
	}

	return nil
}

func DoesDeviceHaveChildren(device string, executor exec.Executor) (bool, error) {
	cmd := fmt.Sprintf("check children for device %s", device)
	children, err := executor.ExecuteCommandPipeline(
		cmd,
		fmt.Sprintf(`lsblk --all -n -l --output PKNAME | grep "^%s$" | awk '{print $0}'`, device))
	if err != nil {
		return false, fmt.Errorf("command %s failed: %+v", cmd, err)
	}

	return children != "", nil
}

func ChownForCurrentUser(path string, executor exec.Executor) {
	if currentUser == nil {
		var err error
		currentUser, err = user.Current()
		if err != nil {
			log.Printf("unable to find current user: %+v", err)
			return
		}
	}

	if currentUser != nil {
		cmd := fmt.Sprintf("chown %s", path)
		if err := executor.ExecuteCommand(cmd, "sudo", "chown", "-R",
			fmt.Sprintf("%s:%s", currentUser.Username, currentUser.Username), path); err != nil {
			log.Printf("command %s failed: %+v", cmd, err)
		}
	}
}
