// Copyright 2018 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package main

import (
	"fmt"

	"github.com/attic-labs/noms/cmd/util"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/spec"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/random"

	"gopkg.in/alecthomas/kingpin.v2"
)

func nomsStruct(noms *kingpin.Application) (*kingpin.CmdClause, util.KingpinHandler) {
	strukt := noms.Command("struct", "interact with structs")

	struktNew := strukt.Command("new", "creates a new struct")
	newDb := struktNew.Arg("database", "spec to db to create struct within").Required().String()
	newName := struktNew.Flag("name", "name for new struct").String()
	newFields := struktNew.Arg("fields", "key/value pairs for field names and values").Strings()

	struktSet := strukt.Command("set", "sets one or more fields of a struct")
	setSpec := struktSet.Arg("spec", "value spec for the struct to edit").Required().String()
	setFields := struktSet.Arg("fields", "key/value pairs for field names and values").Strings()

	return strukt, func(input string) int {
		switch input {
		case struktNew.FullCommand():
			return nomsStructNew(*newDb, *newName, *newFields)
		case struktSet.FullCommand():
			return nomsStructSet(*setSpec, *setFields)
		}
		d.Panic("notreached")
		return 1
	}
}

func nomsStructNew(dbStr string, name string, args []string) int {
	sp, err := spec.ForDatabase(dbStr)
	d.PanicIfError(err)

	if len(args)%2 != 0 {
		d.CheckError(fmt.Errorf("Must be an even number of key/value pairs"))
	}

	db := sp.GetDatabase()
	st := types.NewStruct(name, nil)

	applyStructEdits(db, st, args)
	return 0
}

func nomsStructSet(specStr string, args []string) int {
	sp, err := spec.ForPath(specStr)
	d.PanicIfError(err)

	if len(args)%2 != 0 {
		d.CheckError(fmt.Errorf("Must be an even number of key/value pairs"))
	}

	db := sp.GetDatabase()
	val := sp.GetValue()
	if st, ok := val.(types.Struct); ok {
		applyStructEdits(db, st, args)
		return 0
	} else {
		d.CheckError(fmt.Errorf("Path does not resolve to a struct: %s", specStr))
		return 1
	}
}

func applyStructEdits(db datas.Database, st types.Struct, args []string) {
	for i := 0; i < len(args); i += 2 {
		if !types.IsValidStructFieldName(args[i]) {
			d.CheckError(fmt.Errorf("Invalid field name: %s at position: %d", args[i], i))
		}
		ap, err := spec.NewAbsolutePath(args[i+1])
		if err != nil {
			d.CheckError(fmt.Errorf("Invalid field value: %s at position %d: %s", args[i+1], i+1, err))
		}
		v := ap.Resolve(db)
		if v == nil {
			d.CheckError(fmt.Errorf("Invalid field value: %s at position: %d", args[i+1], i+1))
		}
		st = st.Set(args[i], v)
	}

	// TODO: Committing to this temporary dataset is ghetto, but we lack any
	// other way to flush changes.
	// See: https://github.com/attic-labs/noms/issues/3530
	ds := db.GetDataset(fmt.Sprintf("nomscmd/struct/new/%s", random.Id()))
	r := db.WriteValue(st)
	ds, err := db.CommitValue(ds, r)
	d.PanicIfError(err)
	_, err = db.Delete(ds)
	d.PanicIfError(err)
	fmt.Println(r.TargetHash().String())
}
