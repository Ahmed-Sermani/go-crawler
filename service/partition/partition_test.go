package partition

import (
	"net"
	"os"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
	gc "gopkg.in/check.v1"
)

var _ = gc.Suite(new(RangeTestSuite))

type RangeTestSuite struct{}

func (s *RangeTestSuite) TestNewRangeErrors(c *gc.C) {
	_, err := NewRange(
		uuid.MustParse("40000000-0000-0000-0000-000000000000"),
		uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		1,
	)
	c.Assert(err, gc.ErrorMatches, "range start UUID must be less than the end UUID")

	_, err = NewRange(
		uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		uuid.MustParse("40000000-0000-0000-0000-000000000000"),
		0,
	)
	c.Assert(err, gc.ErrorMatches, "number of partitions must be at least equal to 1")
}

func (s *RangeTestSuite) TestEvenSplit(c *gc.C) {
	r, err := NewFullRange(4)
	c.Assert(err, gc.IsNil)

	expExtents := [][2]uuid.UUID{

		{uuid.MustParse("00000000-0000-0000-0000-000000000000"), uuid.MustParse("40000000-0000-0000-0000-000000000000")},
		{uuid.MustParse("40000000-0000-0000-0000-000000000000"), uuid.MustParse("80000000-0000-0000-0000-000000000000")},
		{uuid.MustParse("80000000-0000-0000-0000-000000000000"), uuid.MustParse("c0000000-0000-0000-0000-000000000000")},
		{uuid.MustParse("c0000000-0000-0000-0000-000000000000"), uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")},
	}

	for i, exp := range expExtents {
		c.Logf("extent: %d", i)
		gotFrom, gotTo, err := r.PartitionExtents(i)
		c.Assert(err, gc.IsNil)
		c.Assert(gotFrom.String(), gc.Equals, exp[0].String())
		c.Assert(gotTo.String(), gc.Equals, exp[1].String())
	}
}

func (s *RangeTestSuite) TestOddSplit(c *gc.C) {
	r, err := NewFullRange(3)
	c.Assert(err, gc.IsNil)

	expExtents := [][2]uuid.UUID{
		{uuid.MustParse("00000000-0000-0000-0000-000000000000"), uuid.MustParse("55555555-5555-5555-5555-555555555555")},
		{uuid.MustParse("55555555-5555-5555-5555-555555555555"), uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")},
		{uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")},
	}

	for i, exp := range expExtents {
		c.Logf("extent: %d", i)
		gotFrom, gotTo, err := r.PartitionExtents(i)
		c.Assert(err, gc.IsNil)
		c.Assert(gotFrom.String(), gc.Equals, exp[0].String())
		c.Assert(gotTo.String(), gc.Equals, exp[1].String())
	}
}

func (s *RangeTestSuite) TestPartitionExtentsError(c *gc.C) {
	r, err := NewRange(
		uuid.MustParse("11111111-0000-0000-0000-000000000000"),
		uuid.MustParse("55555555-0000-0000-0000-000000000000"),
		1,
	)
	c.Assert(err, gc.IsNil)

	_, _, err = r.PartitionExtents(1)
	c.Assert(err, gc.ErrorMatches, "invalid partition index")
}

var _ = gc.Suite(new(DetectorTestSuite))

type DetectorTestSuite struct{}

func (s *DetectorTestSuite) SetUpTest(c *gc.C) {
	getHostname = os.Hostname
	lookupSRV = net.LookupSRV
}

func (s *DetectorTestSuite) TearDownTest(c *gc.C) {
	getHostname = os.Hostname
	lookupSRV = net.LookupSRV
}

func (s *DetectorTestSuite) TestDetectFromSRVRecords(c *gc.C) {
	getHostname = func() (string, error) {
		return "web-1", nil
	}
	lookupSRV = func(service, proto, name string) (string, []*net.SRV, error) {
		c.Assert(service, gc.Equals, "")
		c.Assert(proto, gc.Equals, "")
		c.Assert(name, gc.Equals, "web-service")
		return "web-service", make([]*net.SRV, 4), nil
	}

	det := DetectFromSRVRecords("web-service")
	curPart, numPart, err := det.PartitionInfo()
	c.Assert(err, gc.IsNil)
	c.Assert(curPart, gc.Equals, 1)
	c.Assert(numPart, gc.Equals, 4)
}

func (s *DetectorTestSuite) TestDetectFromSRVRecordsWithNoDataAvailable(c *gc.C) {
	getHostname = func() (string, error) {
		return "web-1", nil
	}
	lookupSRV = func(service, proto, name string) (string, []*net.SRV, error) {
		return "", nil, xerrors.Errorf("host not found")
	}

	det := DetectFromSRVRecords("web-service")
	_, _, err := det.PartitionInfo()
	c.Assert(xerrors.Is(err, ErrNoPartitionDataAvailableYet), gc.Equals, true)
}

func (s *DetectorTestSuite) TestFixedDetector(c *gc.C) {
	det := Fixed{Partition: 1, NumPartitions: 4}

	curPart, numPart, err := det.PartitionInfo()
	c.Assert(err, gc.IsNil)
	c.Assert(curPart, gc.Equals, 1)
	c.Assert(numPart, gc.Equals, 4)
}

func Test(t *testing.T) {
	gc.TestingT(t)
}
