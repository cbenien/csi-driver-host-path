/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azureblob

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"io/ioutil"
	"k8s.io/kubernetes/pkg/util/mount"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"github.com/mitchellh/go-ps"
)

type nodeServer struct {
	nodeID string
}

func NewNodeServer(nodeId string) *nodeServer {
	return &nodeServer{
		nodeID: nodeId,
	}
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {

	// Check arguments
	if req.GetVolumeCapability() == nil {
		return nil, status.Error(codes.InvalidArgument, "Volume capability missing in request")
	}
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	stagingTargetPath := req.GetStagingTargetPath()
	targetPath := req.GetTargetPath()

	err := os.MkdirAll(targetPath, 0750)
	if err != nil {
		return nil, status.Error(codes.Internal, "Mkdir failed")
	}

	args := []string{
		stagingTargetPath,
		targetPath,
		"-o",
		"bind",
	}

	command := "mount"
	cmd := exec.Command(command, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error mount command: %s\nargs: %s\noutput: %s", command, args, out))
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {

	// Check arguments
	if len(req.GetVolumeId()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(req.GetTargetPath()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	targetPath := req.GetTargetPath()

	args := []string{
		targetPath,
	}

	command := "umount"
	cmd := exec.Command(command, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error umount command: %s\nargs: %s\noutput: %s", command, args, out))
	}

	err = os.RemoveAll(targetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to delete %s", targetPath))
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {

	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()

	// Check arguments
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}

	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if req.VolumeCapability == nil {
		return nil, status.Error(codes.InvalidArgument, "NodeStageVolume Volume Capability must be provided")
	}

	if _, err := os.Stat("stagingTargetPath"); !os.IsNotExist(err) {
		// directory exists, assume it's already mounted
		return &csi.NodeStageVolumeResponse{}, nil
	}

	err := os.MkdirAll(stagingTargetPath, 0750)
	if err != nil {
		return nil, status.Error(codes.Internal, "Mkdir failed")
	}

	//	rclone mount --daemon --azureblob-account cbenienblobtest --azureblob-key a2pZKZMswNputBuzuYFvUxjw06rX/EcIeDNqdKIs2FEFW/DqrDc6SpOsySnPaskdXJacr18n7KQToNgaWXeAkQ== --vfs-cache-mode writes :azureblob:test/vol1 /blobby

	storageAccount := "cbenienblobtest"
	accessKey := "a2pZKZMswNputBuzuYFvUxjw06rX/EcIeDNqdKIs2FEFW/DqrDc6SpOsySnPaskdXJacr18n7KQToNgaWXeAkQ=="
	containerName := "test"
	pathPrefix := "vol1"

	args := []string{
		"mount",
		fmt.Sprintf(":azureblob:%s/%s", containerName, pathPrefix),
		fmt.Sprintf("%s", stagingTargetPath),
		fmt.Sprintf("--azureblob-account=%s", storageAccount),
		fmt.Sprintf("--azureblob-key=%s", accessKey),
		"--daemon",
		"--allow-other",
		"--vfs-cache-mode=writes",
	}

	command := "rclone"
	cmd := exec.Command(command, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Error fuseMount command: %s\nargs: %s\noutput: %s", command, args, out))
	}

	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {

	volumeID := req.GetVolumeId()
	stagingTargetPath := req.GetStagingTargetPath()

	// Check arguments
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume ID missing in request")
	}
	if len(stagingTargetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path missing in request")
	}

	if err := mount.New("").Unmount(stagingTargetPath); err != nil {
		return nil, err
	}

	command := "rclone"
	process, err := findFuseMountProcess(stagingTargetPath, command)

	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to find fuse process")
	}

	err = waitForProcess(process, 0)
	if err != nil {
		return nil, status.Error(codes.Internal, "Fuse process didn't exit")
	}

	err = os.RemoveAll(stagingTargetPath)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to delete %s", stagingTargetPath))
	}

	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {

	return &csi.NodeGetInfoResponse{
		NodeId: ns.nodeID,
	}, nil
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_STAGE_UNSTAGE_VOLUME,
					},
				},
			},
		},
	}, nil
}

func (ns *nodeServer) NodeGetVolumeStats(ctx context.Context, in *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func waitForProcess(p *os.Process, backoff int) error {
	if backoff == 20 {
		return fmt.Errorf("Timeout waiting for PID %v to end", p.Pid)
	}
	cmdLine, err := getCmdLine(p.Pid)
	if err != nil {
		glog.Warningf("Error checking cmdline of PID %v, assuming it is dead: %s", p.Pid, err)
		return nil
	}
	if cmdLine == "" {
		// ignore defunct processes
		// TODO: debug why this happens in the first place
		// seems to only happen on k8s, not on local docker
		glog.Warning("Fuse process seems dead, returning")
		return nil
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		glog.Warningf("Fuse process does not seem active or we are unprivileged: %s", err)
		return nil
	}
	glog.Infof("Fuse process with PID %v still active, waiting...", p.Pid)
	time.Sleep(time.Duration(backoff*100) * time.Millisecond)
	return waitForProcess(p, backoff+1)
}

func getCmdLine(pid int) (string, error) {
	cmdLineFile := fmt.Sprintf("/proc/%v/cmdline", pid)
	cmdLine, err := ioutil.ReadFile(cmdLineFile)
	if err != nil {
		return "", err
	}
	return string(cmdLine), nil
}

func findFuseMountProcess(path string, name string) (*os.Process, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	for _, p := range processes {
		if strings.Contains(p.Executable(), name) {
			cmdLine, err := getCmdLine(p.Pid())
			if err != nil {
				glog.Errorf("Unable to get cmdline of PID %v: %s", p.Pid(), err)
				continue
			}
			if strings.Contains(cmdLine, path) {
				glog.Infof("Found matching pid %v on path %s", p.Pid(), path)
				return os.FindProcess(p.Pid())
			}
		}
	}
	return nil, nil
}