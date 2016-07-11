// Copyright 2016 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package libkbfs

import (
	"sync"

	"golang.org/x/net/context"
)

// KBFSStatus represents the content of the top-level status file. It is
// suitable for encoding directly as JSON.
// TODO: implement magical status update like FolderBranchStatus
type KBFSStatus struct {
	CurrentUser     string
	IsConnected     bool
	UsageBytes      int64
	LimitBytes      int64
	FailingServices map[string]error
}

// folderBranchStatusKeeper holds and updates the status for a given
// folder-branch, and produces FolderBranchStatus instances suitable
// for callers outside this package to consume.
type folderBranchStatusKeeper struct {
	config    IFCERFTConfig
	nodeCache IFCERFTNodeCache

	md         *IFCERFTRootMetadata
	dirtyNodes map[IFCERFTNodeID]IFCERFTNode
	unmerged   *IFCERFTCrChains
	merged     *IFCERFTCrChains
	dataMutex  sync.Mutex

	updateChan  chan IFCERFTStatusUpdate
	updateMutex sync.Mutex
}

func newFolderBranchStatusKeeper(
	config IFCERFTConfig, nodeCache IFCERFTNodeCache) *folderBranchStatusKeeper {
	return &folderBranchStatusKeeper{
		config:     config,
		nodeCache:  nodeCache,
		dirtyNodes: make(map[IFCERFTNodeID]IFCERFTNode),
		updateChan: make(chan IFCERFTStatusUpdate, 1),
	}
}

// dataMutex should be taken by the caller
func (fbsk *folderBranchStatusKeeper) signalChangeLocked() {
	fbsk.updateMutex.Lock()
	defer fbsk.updateMutex.Unlock()
	close(fbsk.updateChan)
	fbsk.updateChan = make(chan IFCERFTStatusUpdate, 1)
}

// setRootMetadata sets the current head metadata for the
// corresponding folder-branch.
func (fbsk *folderBranchStatusKeeper) setRootMetadata(md *IFCERFTRootMetadata) {
	fbsk.dataMutex.Lock()
	defer fbsk.dataMutex.Unlock()
	if fbsk.md == md {
		return
	}
	fbsk.md = md
	fbsk.signalChangeLocked()
}

func (fbsk *folderBranchStatusKeeper) setCRChains(unmerged *IFCERFTCrChains, merged *IFCERFTCrChains) {
	fbsk.dataMutex.Lock()
	defer fbsk.dataMutex.Unlock()
	if unmerged == fbsk.unmerged && merged == fbsk.merged {
		return
	}
	fbsk.unmerged = unmerged
	fbsk.merged = merged
	fbsk.signalChangeLocked()
}

func (fbsk *folderBranchStatusKeeper) addNode(m map[IFCERFTNodeID]IFCERFTNode, n IFCERFTNode) {
	fbsk.dataMutex.Lock()
	defer fbsk.dataMutex.Unlock()
	id := n.GetID()
	_, ok := m[id]
	if ok {
		return
	}
	m[id] = n
	fbsk.signalChangeLocked()
}

func (fbsk *folderBranchStatusKeeper) rmNode(m map[IFCERFTNodeID]IFCERFTNode, n IFCERFTNode) {
	fbsk.dataMutex.Lock()
	defer fbsk.dataMutex.Unlock()
	id := n.GetID()
	_, ok := m[id]
	if !ok {
		return
	}
	delete(m, id)
	fbsk.signalChangeLocked()
}

func (fbsk *folderBranchStatusKeeper) addDirtyNode(n IFCERFTNode) {
	fbsk.addNode(fbsk.dirtyNodes, n)
}

func (fbsk *folderBranchStatusKeeper) rmDirtyNode(n IFCERFTNode) {
	fbsk.rmNode(fbsk.dirtyNodes, n)
}

// dataMutex should be taken by the caller
func (fbsk *folderBranchStatusKeeper) convertNodesToPathsLocked(
	m map[IFCERFTNodeID]IFCERFTNode) []string {
	var ret []string
	for _, n := range m {
		ret = append(ret, fbsk.nodeCache.PathFromNode(n).String())
	}
	return ret
}

// getStatus returns a FolderBranchStatus-representation of the
// current status.
func (fbsk *folderBranchStatusKeeper) getStatus(ctx context.Context) (
	IFCERFTFolderBranchStatus, <-chan IFCERFTStatusUpdate, error) {
	fbsk.dataMutex.Lock()
	defer fbsk.dataMutex.Unlock()
	fbsk.updateMutex.Lock()
	defer fbsk.updateMutex.Unlock()

	var fbs IFCERFTFolderBranchStatus

	if fbsk.md != nil {
		fbs.Staged = (fbsk.md.WFlags & IFCERFTMetadataFlagUnmerged) != 0
		name, err := fbsk.config.KBPKI().GetNormalizedUsername(ctx, fbsk.md.LastModifyingWriter)
		if err != nil {
			return IFCERFTFolderBranchStatus{}, nil, err
		}
		fbs.HeadWriter = name
		fbs.DiskUsage = fbsk.md.DiskUsage
		fbs.RekeyPending = fbsk.config.RekeyQueue().IsRekeyPending(fbsk.md.ID)
		fbs.FolderID = fbsk.md.ID.String()
	}

	fbs.DirtyPaths = fbsk.convertNodesToPathsLocked(fbsk.dirtyNodes)

	// Make the chain summaries.  Identify using the unmerged chains,
	// since those are most likely to be able to identify a node in
	// the cache.
	if fbsk.unmerged != nil {
		fbs.Unmerged = fbsk.unmerged.Summary(fbsk.unmerged, fbsk.nodeCache)
		if fbsk.merged != nil {
			fbs.Merged = fbsk.merged.Summary(fbsk.unmerged, fbsk.nodeCache)
		}
	}

	return fbs, fbsk.updateChan, nil
}
