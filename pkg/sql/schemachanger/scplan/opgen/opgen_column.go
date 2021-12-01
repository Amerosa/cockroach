// Copyright 2021 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package opgen

import (
	"github.com/cockroachdb/cockroach/pkg/sql/schemachanger/scop"
	"github.com/cockroachdb/cockroach/pkg/sql/schemachanger/scpb"
)

func init() {
	opRegistry.register((*scpb.Column)(nil),
		add(
			to(scpb.Status_DELETE_ONLY,
				minPhase(scop.PreCommitPhase),
				emit(func(this *scpb.Column) scop.Op {
					return &scop.MakeAddedColumnDeleteOnly{
						TableID:                           this.TableID,
						ColumnID:                          this.ColumnID,
						FamilyName:                        this.FamilyName,
						FamilyID:                          this.FamilyID,
						ColumnType:                        this.Type,
						Nullable:                          this.Nullable,
						DefaultExpr:                       this.DefaultExpr,
						OnUpdateExpr:                      this.OnUpdateExpr,
						Hidden:                            this.Hidden,
						Inaccessible:                      this.Inaccessible,
						GeneratedAsIdentityType:           this.GeneratedAsIdentityType,
						GeneratedAsIdentitySequenceOption: this.GeneratedAsIdentitySequenceOption,
						UsesSequenceIds:                   this.UsesSequenceIds,
						ComputerExpr:                      this.ComputerExpr,
						PgAttributeNum:                    this.PgAttributeNum,
						SystemColumnKind:                  this.SystemColumnKind,
						Virtual:                           this.Virtual,
					}
				}),
				emit(func(this *scpb.Column, md *scpb.ElementMetadata) scop.Op {
					return &scop.LogEvent{Metadata: *md,
						DescID:    this.TableID,
						Element:   &scpb.ElementProto{Column: this},
						Direction: scpb.Target_ADD,
					}
				}),
			),
			to(scpb.Status_DELETE_AND_WRITE_ONLY,
				minPhase(scop.PostCommitPhase),
				emit(func(this *scpb.Column) scop.Op {
					return &scop.MakeAddedColumnDeleteAndWriteOnly{
						TableID:  this.TableID,
						ColumnID: this.ColumnID,
					}
				}),
			),
			to(scpb.Status_PUBLIC,
				emit(func(this *scpb.Column) scop.Op {
					return &scop.MakeColumnPublic{
						TableID:  this.TableID,
						ColumnID: this.ColumnID,
					}
				}),
			),
		),
		drop(
			to(scpb.Status_DELETE_AND_WRITE_ONLY,
				emit(func(this *scpb.Column) scop.Op {
					return &scop.MakeDroppedColumnDeleteAndWriteOnly{
						TableID:  this.TableID,
						ColumnID: this.ColumnID,
					}
				}),
				emit(func(this *scpb.Column, md *scpb.ElementMetadata) scop.Op {
					return &scop.LogEvent{Metadata: *md,
						DescID:    this.TableID,
						Element:   &scpb.ElementProto{Column: this},
						Direction: scpb.Target_DROP,
					}
				}),
			),
			to(scpb.Status_DELETE_ONLY,
				revertible(false),
				minPhase(scop.PostCommitPhase),
				emit(func(this *scpb.Column) scop.Op {
					return &scop.MakeDroppedColumnDeleteOnly{
						TableID:  this.TableID,
						ColumnID: this.ColumnID,
					}
				}),
			),
			to(scpb.Status_ABSENT,
				minPhase(scop.PostCommitPhase),
				emit(func(this *scpb.Column) scop.Op {
					return &scop.MakeColumnAbsent{
						TableID:  this.TableID,
						ColumnID: this.ColumnID,
					}
				}),
			),
		),
	)
}
