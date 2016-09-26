// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"path"
	"testing"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/testify/suite"
)

func TestSync(t *testing.T) {
	suite.Run(t, &nomsSyncTestSuite{})
}

type nomsSyncTestSuite struct {
	clienttest.ClientTestSuite
}

func (s *nomsSyncTestSuite) TestSyncValidation() {
	sourceDB := datas.NewDatabase(chunks.NewLevelDBStore(s.LdbDir, "", 1, false))
	source1 := sourceDB.GetDataset("src")
	source1, err := sourceDB.CommitValue(source1, types.Number(42))
	s.NoError(err)
	source1HeadRef := source1.Head().Hash()
	source1.Database().Close()
	sourceSpecMissingHashSymbol := spec.CreateValueSpecString("ldb", s.LdbDir, source1HeadRef.String())

	ldb2dir := path.Join(s.TempDir, "ldb2")
	sinkDatasetSpec := spec.CreateValueSpecString("ldb", ldb2dir, "dest")

	defer func() {
		err := recover()
		s.Equal(clienttest.ExitError{-1}, err)
	}()

	s.MustRun(main, []string{"sync", sourceSpecMissingHashSymbol, sinkDatasetSpec})
}

func (s *nomsSyncTestSuite) TestSync() {
	sourceDB := datas.NewDatabase(chunks.NewLevelDBStore(s.LdbDir, "", 1, false))
	source1 := sourceDB.GetDataset("src")
	source1, err := sourceDB.CommitValue(source1, types.Number(42))
	s.NoError(err)
	source2, err := sourceDB.CommitValue(source1, types.Number(43))
	s.NoError(err)
	source1HeadRef := source1.Head().Hash()
	source2.Database().Close() // Close Database backing both Datasets

	sourceSpec := spec.CreateValueSpecString("ldb", s.LdbDir, "#"+source1HeadRef.String())
	ldb2dir := path.Join(s.TempDir, "ldb2")
	sinkDatasetSpec := spec.CreateValueSpecString("ldb", ldb2dir, "dest")
	sout, _ := s.MustRun(main, []string{"sync", sourceSpec, sinkDatasetSpec})

	s.Regexp("Created", sout)
	db := datas.NewDatabase(chunks.NewLevelDBStore(ldb2dir, "", 1, false))
	dest := db.GetDataset("dest")
	s.True(types.Number(42).Equals(dest.HeadValue()))
	db.Close()

	sourceDataset := spec.CreateValueSpecString("ldb", s.LdbDir, "src")
	sout, _ = s.MustRun(main, []string{"sync", sourceDataset, sinkDatasetSpec})
	s.Regexp("Synced", sout)

	db = datas.NewDatabase(chunks.NewLevelDBStore(ldb2dir, "", 1, false))
	dest = db.GetDataset("dest")
	s.True(types.Number(43).Equals(dest.HeadValue()))
	db.Close()

	sout, _ = s.MustRun(main, []string{"sync", sourceDataset, sinkDatasetSpec})
	s.Regexp("up to date", sout)
}

func (s *nomsSyncTestSuite) TestRewind() {
	var err error
	sourceDB := datas.NewDatabase(chunks.NewLevelDBStore(s.LdbDir, "", 1, false))
	source1 := sourceDB.GetDataset("foo")
	source1, err = sourceDB.CommitValue(source1, types.Number(42))
	s.NoError(err)
	rewindRef := source1.HeadRef().TargetHash()
	source1, err = sourceDB.CommitValue(source1, types.Number(43))
	s.NoError(err)
	source1.Database().Close() // Close Database backing both Datasets

	sourceSpec := spec.CreateValueSpecString("ldb", s.LdbDir, "#"+rewindRef.String())
	sinkDatasetSpec := spec.CreateValueSpecString("ldb", s.LdbDir, "foo")
	s.MustRun(main, []string{"sync", sourceSpec, sinkDatasetSpec})

	db := datas.NewDatabase(chunks.NewLevelDBStore(s.LdbDir, "", 1, false))
	dest := db.GetDataset("foo")
	s.True(types.Number(42).Equals(dest.HeadValue()))
	db.Close()
}
