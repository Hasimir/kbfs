// Copyright 2018 Keybase Inc. All rights reserved.
// Use of this source code is governed by a BSD
// license that can be found in the LICENSE file.

package kbfsedits

import (
	"time"
)

type NotificationVersion int

const (
	NotificationV1 NotificationVersion = 1
	NotificationV2 NotificationVersion = 2
)

type NotificationOpType string

const (
	NotificationCreate NotificationOpType = "create"
	NotificationModify                    = "modify"
	NotificationRename                    = "rename"
	NotificationDelete                    = "delete"
)

type EntryType string

const (
	EntryTypeFile EntryType = "file"
	EntryTypeExec EntryType = "file"
)

type NotificationMessage struct {
	Version  NotificationVersion
	Filename string
	Type     NotificationOpType
	Time     time.Time // server-reported time

}
