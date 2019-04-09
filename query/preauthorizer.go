package query

import (
	"context"
	"github.com/influxdata/flux/ast"
	platform "github.com/influxdata/influxdb"
	"github.com/pkg/errors"
)

// PreAuthorizer provides a method for ensuring that the buckets accessed by a query spec
// are allowed access by the given Authorization.  This is a pre-check provided as a way for
// callers to fail early for operations that are not allowed.  However, it's still possible
// for authorization to be denied at runtime even if this check passes.
type PreAuthorizer interface {
	PreAuthorize(ctx context.Context, ast *ast.Package, auth platform.Authorizer, orgID *platform.ID) error
	RequiredPermissions(ctx context.Context, ast *ast.Package, orgID *platform.ID) ([]platform.Permission, error)
}

// NewPreAuthorizer creates a new PreAuthorizer
func NewPreAuthorizer(bucketService platform.BucketService) PreAuthorizer {
	return &preAuthorizer{bucketService: bucketService}
}

type preAuthorizer struct {
	bucketService platform.BucketService
}

// PreAuthorize finds all the buckets read and written by the given spec, and ensures that execution is allowed
// given the Authorizer.  Returns nil on success, and an error with an appropriate message otherwise.
func (a *preAuthorizer) PreAuthorize(ctx context.Context, ast *ast.Package, auth platform.Authorizer, orgID *platform.ID) error {
	// TODO(cwolff): re-enable the ability to pre-authorize by determining the buckets accessed by a Flux script
	//  See https://github.com/influxdata/influxdb/issues/13278
	readBuckets := make([]platform.BucketFilter, 0)
	writeBuckets := make([]platform.BucketFilter, 0)

	for _, readBucketFilter := range readBuckets {
		bucket, err := a.bucketService.FindBucket(ctx, readBucketFilter)
		if err != nil {
			return errors.Wrapf(err, "could not find read bucket with filter: %s", readBucketFilter)
		}

		if bucket == nil {
			return errors.New("bucket service returned nil bucket")
		}

		reqPerm, err := platform.NewPermissionAtID(bucket.ID, platform.ReadAction, platform.BucketsResourceType, bucket.OrganizationID)
		if err != nil {
			return errors.Wrapf(err, "could not create read bucket permission")
		}

		if !auth.Allowed(*reqPerm) {
			return errors.New("no read permission for bucket: \"" + bucket.Name + "\"")
		}
	}

	for _, writeBucketFilter := range writeBuckets {
		bucket, err := a.bucketService.FindBucket(ctx, writeBucketFilter)
		if err != nil {
			return errors.Wrapf(err, "could not find write bucket with filter: %s", writeBucketFilter)
		}

		reqPerm, err := platform.NewPermissionAtID(bucket.ID, platform.WriteAction, platform.BucketsResourceType, bucket.OrganizationID)
		if err != nil {
			return errors.Wrapf(err, "could not create write bucket permission")
		}
		if !auth.Allowed(*reqPerm) {
			return errors.New("no write permission for bucket: \"" + bucket.Name + "\"")
		}
	}

	return nil
}

// RequiredPermissions returns a slice of permissions required for the query contained in spec.
// This method also validates that the buckets exist.
func (a *preAuthorizer) RequiredPermissions(ctx context.Context, ast *ast.Package, orgID *platform.ID) ([]platform.Permission, error) {
	// TODO(cwolff): re-enable the ability to pre-authorize by determining the buckets accessed by a Flux script
	//  See https://github.com/influxdata/influxdb/issues/13278
	readBuckets := make([]platform.BucketFilter, 0)
	writeBuckets := make([]platform.BucketFilter, 0)
	ps := make([]platform.Permission, 0, len(readBuckets)+len(writeBuckets))
	for _, readBucketFilter := range readBuckets {
		bucket, err := a.bucketService.FindBucket(ctx, readBucketFilter)
		if err != nil {
			return nil, errors.Wrapf(err, "could not find read bucket with filter: %s", readBucketFilter)
		}

		if bucket == nil {
			return nil, errors.New("bucket service returned nil bucket")
		}

		reqPerm, err := platform.NewPermissionAtID(bucket.ID, platform.ReadAction, platform.BucketsResourceType, bucket.OrganizationID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create read bucket permission")
		}

		ps = append(ps, *reqPerm)
	}

	for _, writeBucketFilter := range writeBuckets {
		bucket, err := a.bucketService.FindBucket(ctx, writeBucketFilter)
		if err != nil {
			return nil, errors.Wrapf(err, "could not find write bucket with filter: %s", writeBucketFilter)
		}

		reqPerm, err := platform.NewPermissionAtID(bucket.ID, platform.WriteAction, platform.BucketsResourceType, bucket.OrganizationID)
		if err != nil {
			return nil, errors.Wrapf(err, "could not create write bucket permission")
		}
		ps = append(ps, *reqPerm)
	}

	return ps, nil
}
