package querytest

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/influxdata/flux"
	"github.com/influxdata/flux/semantic/semantictest"
	"github.com/influxdata/flux/stdlib/universe"
	platform "github.com/influxdata/influxdb"
)

type BucketAwareQueryTestCase struct {
	Name             string
	Raw              string
	Want             *flux.Spec
	WantErr          bool
	WantReadBuckets  *[]platform.BucketFilter
	WantWriteBuckets *[]platform.BucketFilter
}

var opts = append(
	semantictest.CmpOptions,
	cmp.AllowUnexported(flux.Spec{}),
	cmp.AllowUnexported(universe.JoinOpSpec{}),
	cmpopts.IgnoreUnexported(flux.Spec{}),
	cmpopts.IgnoreUnexported(universe.JoinOpSpec{}),
)

func BucketAwareQueryTestHelper(t *testing.T, tc BucketAwareQueryTestCase) {
	t.Skip("BucketsAccessed needs re-implementing; see https://github.com/influxdata/influxdb/issues/13278")
	t.Helper()
	verifyBuckets(nil, nil)
}

func verifyBuckets(wantBuckets, gotBuckets []platform.BucketFilter) string {
	if len(wantBuckets) != len(gotBuckets) {
		return fmt.Sprintf("Expected %v buckets but got %v", len(wantBuckets), len(gotBuckets))
	}

	for i, wantBucket := range wantBuckets {
		if diagnostic := cmp.Diff(wantBucket, gotBuckets[i]); diagnostic != "" {
			return fmt.Sprintf("Bucket mismatch: -want/+got:\n%v", diagnostic)
		}
	}

	return ""
}
