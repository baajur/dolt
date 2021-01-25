// Copyright 2020 Dolthub, Inc.
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

package dfunctions

import (
	"context"
	"errors"
	"fmt"
	"github.com/dolthub/dolt/go/cmd/dolt/cli"
	"github.com/dolthub/dolt/go/libraries/doltcore/doltdb"
	"github.com/dolthub/dolt/go/libraries/doltcore/env"
	"github.com/dolthub/dolt/go/libraries/doltcore/env/actions"
	"github.com/dolthub/dolt/go/libraries/doltcore/sqle"
	"github.com/dolthub/dolt/go/libraries/utils/argparser"
	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"strings"
)

const DoltCheckoutFuncName = "dolt_checkout"

type DoltCheckoutFunc struct {
	expression.NaryExpression
}

func (d DoltCheckoutFunc) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	dbName := ctx.GetCurrentDatabase()

	if len(dbName) == 0 {
		return 1, fmt.Errorf("Empty database name.")
	}

	dSess := sqle.DSessFromSess(ctx.Session)
	dbData, ok := dSess.GetDbData(dbName)

	if !ok {
		return 1, fmt.Errorf("Could not load database %s", dbName)
	}

	ap := cli.CreateCheckoutArgParser()
	args, err := getDoltArgs(ctx, row, d.Children())

	if err != nil {
		return 1, err
	}

	apr := cli.ParseArgs(ap, args, nil)

	if (apr.Contains(cli.CheckoutCoBranch) && apr.NArg() > 1) || (!apr.Contains(cli.CheckoutCoBranch) && apr.NArg() == 0) {
		return 1, errors.New("Improper usage.")
	}

	// Checking out new branch.
	if newBranch, newBranchOk := apr.GetValue(cli.CheckoutCoBranch); newBranchOk {
		if len(newBranch) == 0 {
			err = errors.New("error: cannot checkout empty string")
		} else {
			err = checkoutNewBranch(ctx, dbData, newBranch, apr)
		}

		if err != nil {
			return 1, err
		}

		return 0, nil
	}

	name := apr.Arg(0)

	if len(name) == 0 {
		return 1, errors.New("error: cannot checkout empty string")
	}

	// Check if user wants to checkout branch.
	if isBranch, err := actions.IsBranch(ctx, dbData.Ddb, name); err != nil {
		return 1, err
	} else if isBranch {
		err = checkoutBranch(ctx, dbData, name)
		if err != nil {
			return 1, err
		}
		return 0, nil
	}

	// Check if user want to checkout table or docs.



	return 0, nil
}

func checkoutNewBranch(ctx context.Context, dbData env.DbData, newBranch string, apr *argparser.ArgParseResults) error {
	startPt := "head"
	if apr.NArg() == 1 {
		startPt = apr.Arg(0)
	}

	err := actions.CreateBranchWithStartPt(ctx, dbData, newBranch, startPt, false)
	if err != nil {
		return err
	}

	return checkoutBranch(ctx, dbData, newBranch)
}

func checkoutBranch(ctx context.Context, dbData env.DbData, branchName string) error {
	err := actions.CheckoutBranch(ctx, dbData, branchName)

	if err != nil {
		if err == doltdb.ErrBranchNotFound {
			return fmt.Errorf("fatal: Branch '%s' not found.", branchName)
		} else if actions.IsRootValUnreachable(err) {
			rt := actions.GetUnreachableRootType(err)
			return fmt.Errorf("error: unable to read the %s", rt.String())
		} else if actions.IsCheckoutWouldOverwrite(err) {
			return errors.New("error: Your local changes to the following tables would be overwritten by checkout")
		} else if err == doltdb.ErrAlreadyOnBranch {
			return fmt.Errorf("Already on branch '%s'", branchName)
		} else {
			return fmt.Errorf("fatal: Unexpected error checking out branch '%s'", branchName)
		}
	}

	return nil
}

func (d DoltCheckoutFunc) String() string {
	childrenStrings := make([]string, len(d.Children()))

	for i, child := range d.Children() {
		childrenStrings[i] = child.String()
	}

	return fmt.Sprintf("DOLT_CHECKOUT(%s)", strings.Join(childrenStrings, ","))
}

// TODO: This might not a branch
func (d DoltCheckoutFunc) Type() sql.Type {
	return sql.Int8
}


func (d DoltCheckoutFunc) WithChildren(children ...sql.Expression) (sql.Expression, error) {
	return NewDoltCheckoutFunc(children...)
}

func NewDoltCheckoutFunc(args ...sql.Expression) (sql.Expression, error) {
	return &DoltCheckoutFunc{expression.NaryExpression{ChildExpressions: args}}, nil
}
