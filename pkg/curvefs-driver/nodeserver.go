/*
Copyright 2022 The Curve Authors

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

package curvefsdriver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/google/uuid"
	"github.com/opencurve/curvefs-csi/pkg/csicommon"
	"github.com/opencurve/curvefs-csi/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
	"k8s.io/utils/mount"
)

type nodeServer struct {
	*csicommon.DefaultNodeServer
	mounter     mount.Interface
	mountRecord map[string]string // targetPath -> a uuid
}

func (ns *nodeServer) NodePublishVolume(
	ctx context.Context,
	req *csi.NodePublishVolumeRequest,
) (*csi.NodePublishVolumeResponse, error) {
	klog.V(5).Infof("%s: called with args %+v", util.GetCurrentFuncName(), *req)
	volumeID := req.GetVolumeId()
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path is missing")
	}
	if !util.ValidateCharacter([]string{targetPath}) {
		return nil, status.Errorf(codes.InvalidArgument, "Illegal TargetPath: %s", targetPath)
	}

	isNotMounted, _ := mount.IsNotMountPoint(ns.mounter, targetPath)
	if !isNotMounted {
		klog.V(5).Infof("%s is already mounted", targetPath)
		return &csi.NodePublishVolumeResponse{}, nil
	}
	err := util.CreatTargetPath(targetPath)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Failed to create target path %s, err: %v",
			targetPath,
			err,
		)
	}

	curvefsMounter := NewCurvefsMounter()
	mountUUID := uuid.New().String()
	err = curvefsMounter.MountFs(volumeID, targetPath, req.GetVolumeContext(),
		req.GetVolumeCapability().GetMount(), mountUUID)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"Failed to mount %s to %s, err: %v",
			volumeID,
			targetPath,
			err,
		)
	}

	isNotMounted, _ = mount.IsNotMountPoint(ns.mounter, targetPath)
	if isNotMounted {
		return nil, status.Errorf(
			codes.Internal,
			"Mount check failed, targetPath: %s",
			targetPath,
		)
	}
	ns.mountRecord[targetPath] = mountUUID
	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(
	ctx context.Context,
	req *csi.NodeUnpublishVolumeRequest,
) (*csi.NodeUnpublishVolumeResponse, error) {
	klog.V(5).Infof("%s: called with args %+v", util.GetCurrentFuncName(), *req)
	targetPath := req.GetTargetPath()
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Target path is missing")
	}
	if !util.ValidateCharacter([]string{targetPath}) {
		return nil, status.Errorf(codes.InvalidArgument, "Illegal TargetPath: %s", targetPath)
	}

	isNotMounted, _ := mount.IsNotMountPoint(ns.mounter, targetPath)
	if isNotMounted {
		klog.V(5).Infof("%s is not mounted", targetPath)
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	curvefsMounter := NewCurvefsMounter()
	mountUUID := ns.mountRecord[targetPath]
	err := curvefsMounter.UmountFs(targetPath, mountUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal,
			"Failed to umount %s, err: %v",
			targetPath,
			err,
		)
	}

	isNotMounted, _ = mount.IsNotMountPoint(ns.mounter, targetPath)
	if !isNotMounted {
		return nil, status.Errorf(
			codes.Internal,
			"Umount check failed, targetPath: %s",
			targetPath,
		)
	}
	delete(ns.mountRecord, targetPath)
	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetInfo(
	ctx context.Context,
	req *csi.NodeGetInfoRequest,
) (*csi.NodeGetInfoResponse, error) {
	klog.V(5).Infof("%s: called with args %+v", util.GetCurrentFuncName(), *req)
	return &csi.NodeGetInfoResponse{
		NodeId: ns.Driver.NodeID,
	}, nil
}

func (ns *nodeServer) NodeGetCapabilities(
	ctx context.Context,
	req *csi.NodeGetCapabilitiesRequest,
) (*csi.NodeGetCapabilitiesResponse, error) {
	klog.V(5).Infof("%s: called with args %+v", util.GetCurrentFuncName(), *req)
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{},
	}, nil
}
