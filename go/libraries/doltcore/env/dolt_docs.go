// Copyright 2019 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package env

import (
	"context"
	"errors"

	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdocs"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle/docsTable"
)

// ResetWorkingDocsToStagedDocs resets the `dolt_docs` table on the working root to match the staged root.
// If the `dolt_docs` table does not exist on the staged root, it will be removed from the working root.
func ResetWorkingDocsToStagedDocs(ctx context.Context, ddb *doltdb.DoltDB, rsr RepoStateReader, rsw RepoStateWriter) error {
	wrkRoot, err := WorkingRoot(ctx, ddb, rsr)
	if err != nil {
		return err
	}

	stgRoot, err := StagedRoot(ctx, ddb, rsr)
	if err != nil {
		return err
	}

	stgDocTbl, stgDocsFound, err := stgRoot.GetTable(ctx, doltdb.DocTableName)
	if err != nil {
		return err
	}

	_, wrkDocsFound, err := wrkRoot.GetTable(ctx, doltdb.DocTableName)
	if err != nil {
		return err
	}

	if wrkDocsFound && !stgDocsFound {
		newWrkRoot, err := wrkRoot.RemoveTables(ctx, doltdb.DocTableName)
		if err != nil {
			return err
		}
		_, err = UpdateWorkingRoot(ctx, ddb, rsw, newWrkRoot)
		return err
	}

	if stgDocsFound {
		newWrkRoot, err := wrkRoot.PutTable(ctx, doltdb.DocTableName, stgDocTbl)
		if err != nil {
			return err
		}
		_, err = UpdateWorkingRoot(ctx, ddb, rsw, newWrkRoot)
		return err
	}
	return nil
}

// UpdateRootWithDocs takes in a root value, a drw, and some docs and writes those docs to the dolt_docs table
// (perhaps creating it in the process). The table might not necessarily need to be created if there are no docs in the
// repo yet.
func UpdateRootWithDocs(ctx context.Context, dbData DbData, root *doltdb.RootValue, rootType RootType, docs doltdocs.Docs) (*doltdb.RootValue, error) {
	docTbl, _, err := root.GetTable(ctx, doltdb.DocTableName)

	if err != nil {
		return nil, err
	}

	docTbl, err = docsTable.CreateOrUpdateDocsTable(ctx, root.VRW(), docs, docTbl)

	if errors.Is(docsTable.ErrEmptyDocsTable, err) {
		root, err = root.RemoveTables(ctx, doltdb.DocTableName)
	} else if err != nil {
		return nil, err
	}

	// There might not need be a need to create docs table if not docs have been created yet so check if docTbl != nil.
	if docTbl != nil {
		root, err = root.PutTable(ctx, doltdb.DocTableName, docTbl)
	}

	switch rootType {
	case Working:
		_, err = UpdateWorkingRoot(ctx, dbData.Ddb, dbData.Rsw, root)
	case Staged:
		_, err = UpdateStagedRoot(ctx, dbData.Ddb, dbData.Rsw, root)
	default:
		return nil, errors.New("Root type not supported with docs update.")
	}
	return root, nil
}

func UpdateRootToDocs(ctx context.Context, root *doltdb.RootValue, docs doltdocs.Docs) (*doltdb.RootValue, error) {
	docTbl, _, err := root.GetTable(ctx, doltdb.DocTableName)

	if err != nil {
		return nil, err
	}

	docTbl, err = docsTable.CreateOrUpdateDocsTable(ctx, root.VRW(), docs, docTbl)

	if errors.Is(docsTable.ErrEmptyDocsTable, err) {
		root, err = root.RemoveTables(ctx, doltdb.DocTableName)
	} else if err != nil {
		return nil, err
	}

	// There might not need be a need to create docs table if not docs have been created yet so check if docTbl != nil.
	if docTbl != nil {
		root, err = root.PutTable(ctx, doltdb.DocTableName, docTbl)
	}

	return root, nil
}
