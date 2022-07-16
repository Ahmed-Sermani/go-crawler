package partition

import (
	"bytes"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

var (
	getHostname = os.Hostname
	lookupSRV   = net.LookupSRV

	// ErrNoPartitionDataAvailableYet is returned by the SRV-aware
	// partition detector to indicate that SRV records for this target
	// application are not yet available.
	ErrNoPartitionDataAvailableYet = xerrors.Errorf("no partition data available yet")
)

// Range represents a contiguous UUID region which is split into a number of
// partitions.
type Range struct {
	start       uuid.UUID
	rangeSplits []uuid.UUID
}

// NewFullRange creates a new range that uses the full UUID value space and
// splits it into the provided number of partitions.
func NewFullRange(numPartitions int) (Range, error) {
	return NewRange(
		uuid.Nil,
		uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		numPartitions,
	)
}

// NewRange creates a new range [start, end) and splits it into the
// provided number of partitions.
func NewRange(start, end uuid.UUID, numPartitions int) (Range, error) {
	if bytes.Compare(start[:], end[:]) >= 0 {
		return Range{}, xerrors.Errorf("range start UUID must be less than the end UUID")
	} else if numPartitions <= 0 {
		return Range{}, xerrors.Errorf("number of partitions must be at least equal to 1")
	}

	// Calculate the size of each partition as: ((end - start + 1) / numPartitions)
	tokenRange := big.NewInt(0)
	partSize := big.NewInt(0)
	partSize = partSize.Sub(big.NewInt(0).SetBytes(end[:]), big.NewInt(0).SetBytes(start[:]))
	partSize = partSize.Div(partSize.Add(partSize, big.NewInt(1)), big.NewInt(int64(numPartitions)))

	var (
		to     uuid.UUID
		err    error
		ranges = make([]uuid.UUID, numPartitions)
	)
	for partition := 0; partition < numPartitions; partition++ {
		if partition == numPartitions-1 {
			to = end
		} else {
			tokenRange.Mul(partSize, big.NewInt(int64(partition+1)))
			if to, err = uuid.FromBytes(tokenRange.Bytes()); err != nil {
				return Range{}, xerrors.Errorf("partition range: %w", err)
			}
		}

		ranges[partition] = to
	}

	return Range{start: start, rangeSplits: ranges}, nil
}

// PartitionExtents returns the [start, end) range for the requested partition.
func (r Range) PartitionExtents(partition int) (uuid.UUID, uuid.UUID, error) {
	if partition < 0 || partition >= len(r.rangeSplits) {
		return uuid.Nil, uuid.Nil, xerrors.Errorf("invalid partition index")
	}

	if partition == 0 {
		return r.start, r.rangeSplits[0], nil
	}
	return r.rangeSplits[partition-1], r.rangeSplits[partition], nil
}

// Detector is implemented by types that can assign a clustered application
// instance to a particular partition.
type Detector interface {
	PartitionInfo() (int, int, error)
}

// FromSRVRecords detects the number of partitions by performing an SRV query
// and counting the number of results.
type FromSRVRecords struct {
	srvName string
}

// DetectFromSRVRecords returns a PartitionDetector implementation that
// extracts the current partition name from the current host name and attempts
// to detect the total number of partitions by performing an SRV query and
// counting the number of responses.
//
// This detector is meant to be used in conjunction with a Stateful Set in
// a kubernetes environment.
func DetectFromSRVRecords(srvName string) FromSRVRecords {
	return FromSRVRecords{srvName: srvName}
}

// PartitionInfo implements PartitionDetector.
func (det FromSRVRecords) PartitionInfo() (int, int, error) {
	hostname, err := getHostname()
	if err != nil {
		return -1, -1, xerrors.Errorf("partition detector: unable to detect host name: %w", err)
	}
	tokens := strings.Split(hostname, "-")
	partition, err := strconv.ParseInt(tokens[len(tokens)-1], 10, 32)
	if err != nil {
		return -1, -1, xerrors.Errorf("partition detector: unable to extract partition number from host name suffix")
	}

	_, addrs, err := lookupSRV("", "", det.srvName)
	if err != nil {
		return -1, -1, ErrNoPartitionDataAvailableYet
	}

	return int(partition), len(addrs), nil
}

// Fixed is a dummy PartitionDetector implementation that always returns back
// the same partition details.
type Fixed struct {
	// The assigned partition number.
	Partition int

	// The number of partitions.
	NumPartitions int
}

// PartitionInfo implements PartitionDetector.
func (det Fixed) PartitionInfo() (int, int, error) { return det.Partition, det.NumPartitions, nil }
