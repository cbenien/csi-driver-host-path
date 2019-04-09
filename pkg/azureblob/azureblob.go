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
)

const (
	kib    int64 = 1024
	mib    int64 = kib * 1024
	gib    int64 = mib * 1024
	gib100 int64 = gib * 100
	tib    int64 = gib * 1024
	tib100 int64 = tib * 100
)

type azureBlob struct {
	name     string
	nodeID   string
	version  string
	endpoint string

	ids *identityServer
	ns  *nodeServer
	cs  *controllerServer
}

type azureBlobVolume struct {
	VolName       string     `json:"volName"`
	VolID         string     `json:"volID"`
	VolSize       int64      `json:"volSize"`
	VolPath       string     `json:"volPath"`
	VolAccessType accessType `json:"volAccessType"`
}

/*
type azureBlobSnapshot struct {
	Name         string              `json:"name"`
	Id           string              `json:"id"`
	VolID        string              `json:"volID"`
	Path         string              `json:"path"`
	CreationTime timestamp.Timestamp `json:"creationTime"`
	SizeBytes    int64               `json:"sizeBytes"`
	ReadyToUse   bool                `json:"readyToUse"`
}
 */

var azureBlobVolumes map[string]azureBlobVolume
//var hostPathVolumeSnapshots map[string]hostPathSnapshot

var (
	vendorVersion = "dev"
)

func init() {
	azureBlobVolumes = map[string]azureBlobVolume{}
//	hostPathVolumeSnapshots = map[string]hostPathSnapshot{}
}

func NewAzureBlobDriver(driverName, nodeID, endpoint, version string) (*azureBlob, error) {
	if driverName == "" {
		return nil, fmt.Errorf("No driver name provided")
	}

	if nodeID == "" {
		return nil, fmt.Errorf("No node id provided")
	}

	if endpoint == "" {
		return nil, fmt.Errorf("No driver endpoint provided")
	}
	if version != "" {
		vendorVersion = version
	}

	glog.Infof("Driver: %v ", driverName)
	glog.Infof("Version: %s", vendorVersion)

	return &azureBlob{
		name:     driverName,
		version:  vendorVersion,
		nodeID:   nodeID,
		endpoint: endpoint,
	}, nil
}

func (ab *azureBlob) Run() {

	// Create GRPC servers
	ab.ids = NewIdentityServer(ab.name, ab.version)
	ab.ns = NewNodeServer(ab.nodeID)
	ab.cs = NewControllerServer()

	s := NewNonBlockingGRPCServer()
	s.Start(ab.endpoint, ab.ids, ab.cs, ab.ns)
	s.Wait()
}

/*
func getVolumeByID(volumeID string) (azureBlobVolume, error) {
	if hostPathVol, ok := azureBlobVolumes[volumeID]; ok {
		return hostPathVol, nil
	}
	return azureBlobVolume{}, fmt.Errorf("volume id %s does not exit in the volumes list", volumeID)
}

func getVolumeByName(volName string) (azureBlobVolume, error) {
	for _, hostPathVol := range azureBlobVolumes {
		if hostPathVol.VolName == volName {
			return hostPathVol, nil
		}
	}
	return azureBlobVolume{}, fmt.Errorf("volume name %s does not exit in the volumes list", volName)
}

 */

/*
func getSnapshotByName(name string) (hostPathSnapshot, error) {
	for _, snapshot := range hostPathVolumeSnapshots {
		if snapshot.Name == name {
			return snapshot, nil
		}
	}
	return hostPathSnapshot{}, fmt.Errorf("snapshot name %s does not exit in the snapshots list", name)
}
*/