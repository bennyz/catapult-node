package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/containers/image/copy"
	"github.com/containers/image/transports/alltransports"

	"github.com/containers/image/signature"
	ociTools "github.com/opencontainers/image-tools/image"
)

func pullImage(ctx context.Context, imageName string) (string, error) {
	workingDir := fmt.Sprintf("/tmp/%s/", imageName)
	tag := "tmp"
	err := os.Mkdir(workingDir, 0755)
	if err != nil {
		return "", err
	}

	policy := &signature.Policy{Default: []signature.PolicyRequirement{
		signature.NewPRInsecureAcceptAnything(),
	}}

	policyCtx, err := signature.NewPolicyContext(policy)
	if err != nil {
		return "", err
	}

	srcImageType, err :=
		alltransports.ParseImageName(fmt.Sprintf("docker://%s", imageName))
	if err != nil {
		return "", err

	}

	dstImageType, err :=
		alltransports.ParseImageName(fmt.Sprintf("oci:/%s:%s", workingDir, tag))
	if err != nil {
		return "", err
	}
	_, err = copy.Image(ctx,
		policyCtx,
		dstImageType,
		srcImageType,
		&copy.Options{
			ReportWriter: os.Stdout,
		})

	if err != nil {
		return "", err
	}

	ociTools.CreateRuntimeBundleLayout(
		workingDir,
		workingDir,
		"rootfs",
		"",
		[]string{fmt.Sprintf("name=%s", tag)})

	return workingDir + "rootfs", nil
}

func createRootFS(path string) (string, int64, error) {
	size, err := dirSize(path)
	if err != nil {
		return "", -1, err
	}

	err = os.Truncate("rootfs", size)
	if err != nil {
		return "", -1, err
	}

	cmd := exec.Command("mkfs.ext4", "rootfs")
	err = cmd.Run()
	if err != nil {
		return "", -1, err
	}

	cmd = exec.Command("mount", "-o", "loop", "rootfs", path+"/mnt/")
	err = cmd.Run()
	if err != nil {
		return "", -1, err
	}

	cmd = exec.Command("tar", "-c", "xvf", path+"rootfs.tar")
	err = cmd.Run()
	if err != nil {
		return "", -1, err
	}

	cmd = exec.Command("umount", path+"/mnt/")
	err = cmd.Run()
	if err != nil {
		return "", -1, err
	}

	info, err := os.Stat(path + "rootfs.tar")
	if err != nil {
		return "", -1, err
	}

	return path + "rootfs.tar", info.Size(), nil
}

func mapVolume(volumeID, pool string) error {
	conf := "/etc/ceph/ceph.conf"
	keyring := "/etc/ceph/ceph.client.admin.keyring"

	cmd := exec.Command("rbd", "map", fmt.Sprintf("%s/%s", pool, volumeID), "-k", keyring, conf)
	err := cmd.Run()

	return err
}

func dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}
