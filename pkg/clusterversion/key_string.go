// Code generated by "stringer -type=Key"; DO NOT EDIT.

package clusterversion

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[replacedTruncatedAndRangeAppliedStateMigration-0]
	_ = x[replacedPostTruncatedAndRangeAppliedStateMigration-1]
	_ = x[TruncatedAndRangeAppliedStateMigration-2]
	_ = x[PostTruncatedAndRangeAppliedStateMigration-3]
	_ = x[V21_1-4]
	_ = x[Start21_1PLUS-5]
	_ = x[Start21_2-6]
	_ = x[JoinTokensTable-7]
	_ = x[AcquisitionTypeInLeaseHistory-8]
	_ = x[SerializeViewUDTs-9]
	_ = x[ExpressionIndexes-10]
	_ = x[DeleteDeprecatedNamespaceTableDescriptorMigration-11]
	_ = x[FixDescriptors-12]
	_ = x[SQLStatsTable-13]
	_ = x[DatabaseRoleSettings-14]
	_ = x[TenantUsageTable-15]
	_ = x[SQLInstancesTable-16]
	_ = x[NewRetryableRangefeedErrors-17]
	_ = x[AlterSystemWebSessionsCreateIndexes-18]
	_ = x[SeparatedIntentsMigration-19]
	_ = x[PostSeparatedIntentsMigration-20]
	_ = x[RetryJobsWithExponentialBackoff-21]
	_ = x[RecordsBasedRegistry-22]
	_ = x[AutoSpanConfigReconciliationJob-23]
	_ = x[PreventNewInterleavedTables-24]
	_ = x[EnsureNoInterleavedTables-25]
	_ = x[DefaultPrivileges-26]
	_ = x[ZonesTableForSecondaryTenants-27]
	_ = x[UseKeyEncodeForHashShardedIndexes-28]
	_ = x[DatabasePlacementPolicy-29]
	_ = x[GeneratedAsIdentity-30]
	_ = x[OnUpdateExpressions-31]
	_ = x[SpanConfigurationsTable-32]
	_ = x[BoundedStaleness-33]
	_ = x[SQLStatsCompactionScheduledJob-34]
	_ = x[DateAndIntervalStyle-35]
	_ = x[PebbleFormatVersioned-36]
	_ = x[MarkerDataKeysRegistry-37]
	_ = x[PebbleSetWithDelete-38]
	_ = x[TenantUsageSingleConsumptionColumn-39]
	_ = x[V21_2-40]
}

const _Key_name = "replacedTruncatedAndRangeAppliedStateMigrationreplacedPostTruncatedAndRangeAppliedStateMigrationTruncatedAndRangeAppliedStateMigrationPostTruncatedAndRangeAppliedStateMigrationV21_1Start21_1PLUSStart21_2JoinTokensTableAcquisitionTypeInLeaseHistorySerializeViewUDTsExpressionIndexesDeleteDeprecatedNamespaceTableDescriptorMigrationFixDescriptorsSQLStatsTableDatabaseRoleSettingsTenantUsageTableSQLInstancesTableNewRetryableRangefeedErrorsAlterSystemWebSessionsCreateIndexesSeparatedIntentsMigrationPostSeparatedIntentsMigrationRetryJobsWithExponentialBackoffRecordsBasedRegistryAutoSpanConfigReconciliationJobPreventNewInterleavedTablesEnsureNoInterleavedTablesDefaultPrivilegesZonesTableForSecondaryTenantsUseKeyEncodeForHashShardedIndexesDatabasePlacementPolicyGeneratedAsIdentityOnUpdateExpressionsSpanConfigurationsTableBoundedStalenessSQLStatsCompactionScheduledJobDateAndIntervalStylePebbleFormatVersionedMarkerDataKeysRegistryPebbleSetWithDeleteTenantUsageSingleConsumptionColumnV21_2"

var _Key_index = [...]uint16{0, 46, 96, 134, 176, 181, 194, 203, 218, 247, 264, 281, 330, 344, 357, 377, 393, 410, 437, 472, 497, 526, 557, 577, 608, 635, 660, 677, 706, 739, 762, 781, 800, 823, 839, 869, 889, 910, 932, 951, 985, 990}

func (i Key) String() string {
	if i < 0 || i >= Key(len(_Key_index)-1) {
		return "Key(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Key_name[_Key_index[i]:_Key_index[i+1]]
}
