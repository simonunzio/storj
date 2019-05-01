// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	monkit "gopkg.in/spacemonkeygo/monkit.v2"

	"storj.io/storj/pkg/pb"
)

var (
	mon = monkit.Package()

	// the following fields are set by linker flags. if any of them
	// are set and fail to parse, the program will fail to start
	buildTimestamp  string // unix seconds since epoch
	buildCommitHash string
	buildVersion    string // semantic version format
	buildRelease    string // true/false

	// Build is a struct containing all relevant build information associated with the binary
	Build Info
)

// Info is the versioning information for a binary
type Info struct {
	// sync/atomic cache
	commitHashCRC uint32

	Timestamp  time.Time `json:"timestamp,omitempty"`
	CommitHash string    `json:"commitHash,omitempty"`
	Version    SemVer    `json:"version"`
	Release    bool      `json:"release,omitempty"`
}

// SemVer represents a semantic version
type SemVer struct {
	Major int64 `json:"major"`
	Minor int64 `json:"minor"`
	Patch int64 `json:"patch"`
}

// AllowedVersions provides a list of SemVer per Service
type AllowedVersions struct {
	Bootstrap   []SemVer
	Satellite   []SemVer
	Storagenode []SemVer
	Uplink      []SemVer
	Gateway     []SemVer
	Identity    []SemVer
}

// SemVerRegex is the regular expression used to parse a semantic version.
// https://github.com/Masterminds/semver/blob/master/LICENSE.txt
const SemVerRegex string = `v?([0-9]+)\.([0-9]+)\.([0-9]+)`

var versionRegex = regexp.MustCompile("^" + SemVerRegex + "$")

// NewSemVer parses a given version and returns an instance of SemVer or
// an error if unable to parse the version.
func NewSemVer(v string) (*SemVer, error) {
	m := versionRegex.FindStringSubmatch(v)
	if m == nil {
		return nil, errors.New("invalid semantic version for build")
	}

	sv := SemVer{}

	var err error

	// first entry of m is the entire version string
	sv.Major, err = strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return nil, err
	}

	sv.Minor, err = strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		return nil, err
	}

	sv.Patch, err = strconv.ParseInt(m[3], 10, 64)
	if err != nil {
		return nil, err
	}

	return &sv, nil
}

// String converts the SemVer struct to a more easy to handle string
func (sem *SemVer) String() (version string) {
	return fmt.Sprintf("v%d.%d.%d", sem.Major, sem.Minor, sem.Patch)
}

// New creates Version_Info from a json byte array
func New(data []byte) (v Info, err error) {
	err = json.Unmarshal(data, &v)
	return v, err
}

// Marshal converts the existing Version Info to any json byte array
func (v Info) Marshal() (data []byte, err error) {
	data, err = json.Marshal(v)
	return
}

// Proto converts an Info struct to a pb.NodeVersion
// TODO: shouldn't we just use pb.NodeVersion everywhere? gogoproto will let
// us make it match Info.
func (v Info) Proto() (*pb.NodeVersion, error) {
	pbts, err := ptypes.TimestampProto(v.Timestamp)
	if err != nil {
		return nil, err
	}
	return &pb.NodeVersion{
		Version:    v.Version.String(),
		CommitHash: v.CommitHash,
		Timestamp:  pbts,
		Release:    v.Release,
	}, nil
}

// containsVersion compares the allowed version array against the passed version
func containsVersion(all []SemVer, x SemVer) bool {
	for _, n := range all {
		if x == n {
			return true
		}
	}
	return false
}

// StrToSemVerList converts a list of versions to a list of SemVer
func StrToSemVerList(serviceVersions []string) (versions []SemVer, err error) {
	for _, subversion := range serviceVersions {
		sVer, err := NewSemVer(subversion)
		if err != nil {
			return nil, err
		}
		versions = append(versions, *sVer)
	}
	return versions, err
}

func init() {
	if buildVersion == "" && buildTimestamp == "" && buildCommitHash == "" && buildRelease == "" {
		return
	}
	timestamp, err := strconv.ParseInt(buildTimestamp, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid timestamp: %v", err))
	}
	Build = Info{
		Timestamp:  time.Unix(timestamp, 0),
		CommitHash: buildCommitHash,
		Release:    strings.ToLower(buildRelease) == "true",
	}

	sv, err := NewSemVer(buildVersion)
	if err != nil {
		panic(err)
	}

	Build.Version = *sv

	if Build.Timestamp.Unix() == 0 || Build.CommitHash == "" {
		Build.Release = false
	}

}
