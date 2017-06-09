package lvmutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func LvDevicePath(vgname, lvname string) string {
	return fmt.Sprintf("/dev/mapper/%s-%s", vgname, lvname)
}

func TPoolDevicePath(vgname, poolname string) string {
	return fmt.Sprintf("/dev/mapper/%s-%s-tpool", vgname, poolname)
}

func myRunCmd(cmd *exec.Cmd) error {
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %s failed: %s", filepath.Base(cmd.Path), err.Error())
	}

	return nil
}

func CreateThinPool(vgname, poolname string, size, chunksize int64) error {
	path, err := exec.LookPath("lvcreate")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, "--size", fmt.Sprintf("%db", size),
		"--chunksize", fmt.Sprintf("%db", chunksize),
		"--thinpool", fmt.Sprintf("%s/%s", vgname, poolname))
	return myRunCmd(cmd)
}

func CreateThinLv(vgname, poolname, lvname string, size int64) error {
	path, err := exec.LookPath("lvcreate")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, "--virtualsize", fmt.Sprintf("%db", size),
		"--name", lvname, "--thinpool", fmt.Sprintf("%s/%s", vgname, poolname))
	cmd.Stderr = os.Stderr
	return myRunCmd(cmd)
}

func CreateSnapshotLv(vgname, lvname, snapname string) error {
	path, err := exec.LookPath("lvcreate")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, "--snapshot", fmt.Sprintf("%s/%s", vgname, lvname), "--name", snapname)
	cmd.Stderr = os.Stderr
	return myRunCmd(cmd)
}

func ResizeLv(vgname, lvname string, size int64) error {
	path, err := exec.LookPath("lvresize")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, "--size", fmt.Sprintf("%db", size), fmt.Sprintf("%s/%s", vgname, lvname))
	cmd.Stderr = os.Stderr
	return myRunCmd(cmd)
}

func RenameLv(vgname, oldname, newname string) error {
	path, err := exec.LookPath("lvrename")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, vgname, oldname, newname)
	cmd.Stderr = os.Stderr
	return myRunCmd(cmd)
}

func RemoveLv(vgname, lvname string, force bool) error {
	path, err := exec.LookPath("lvremove")
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	if force {
		cmd = exec.Command(path, "-f", fmt.Sprintf("%s/%s", vgname, lvname))
	} else {
		cmd = exec.Command(path, fmt.Sprintf("%s/%s", vgname, lvname))
	}
	cmd.Stderr = os.Stderr
	return myRunCmd(cmd)
}

func ActivateLv(vgname, lvname string) error {
	path, err := exec.LookPath("lvchange")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, "-ay", "-K", fmt.Sprintf("%s/%s", vgname, lvname))
	cmd.Stderr = os.Stderr
	return myRunCmd(cmd)
}

func DeactivateLv(vgname, lvname string) error {
	path, err := exec.LookPath("lvchange")
	if err != nil {
		return err
	}

	cmd := exec.Command(path, "-an", fmt.Sprintf("%s/%s", vgname, lvname))
	cmd.Stderr = os.Stderr
	return myRunCmd(cmd)
}

func SetLvActivationSkip(vgname, lvname string, skip bool) error {
	path, err := exec.LookPath("lvchange")
	if err != nil {
		return err
	}

	var flag string
	if skip {
		flag = "-ky"
	} else {
		flag = "-kn"
	}

	cmd := exec.Command(path, flag, fmt.Sprintf("%s/%s", vgname, lvname))
	cmd.Stderr = os.Stderr
	return myRunCmd(cmd)
}
