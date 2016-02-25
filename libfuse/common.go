package libfuse

import (
	"time"

	"bazil.org/fuse"
	"github.com/keybase/client/go/logger"
	"github.com/keybase/kbfs/libkbfs"
	"golang.org/x/net/context"
)

const (
	// PublicName is the name of the parent of all public top-level folders.
	PublicName = "public"

	// PrivateName is the name of the parent of all private top-level folders.
	PrivateName = "private"

	// CtxAppIDKey is the context app id
	CtxAppIDKey = "kbfsfuse-app-id"

	// CtxOpID is the display name for the unique operation FUSE ID tag.
	CtxOpID = "FID"
)

// CtxTagKey is the type used for unique context tags
type CtxTagKey int

const (
	// CtxIDKey is the type of the tag for unique operation IDs.
	CtxIDKey CtxTagKey = iota
)

// NewContextWithOpID adds a unique ID to this context, identifying
// a particular request.
func NewContextWithOpID(ctx context.Context,
	log logger.Logger) context.Context {
	return ctx
}

// fillAttr sets attributes based on the entry info. It only handles fields
// common to all entryinfo types.
func fillAttr(ei *libkbfs.EntryInfo, a *fuse.Attr) {
	a.Valid = 1 * time.Minute

	a.Size = ei.Size
	a.Mtime = time.Unix(0, ei.Mtime)
	a.Ctime = time.Unix(0, ei.Ctime)
}
