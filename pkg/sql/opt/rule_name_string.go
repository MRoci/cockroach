// Code generated by "stringer -output=rule_name_string.go -type=RuleName rule_name.go rule_name.og.go"; DO NOT EDIT.

package opt

import "strconv"

const _RuleName_name = "InvalidRuleNameNumManualRuleNamesEliminateEmptyAndEliminateEmptyOrEliminateSingletonAndOrSimplifyAndSimplifyOrSimplifyFiltersFoldNullAndOrNegateComparisonEliminateNotNegateAndNegateOrCommuteVarInequalityCommuteConstInequalityNormalizeCmpPlusConstNormalizeCmpMinusConstNormalizeCmpConstMinusNormalizeTupleEqualityFoldNullComparisonLeftFoldNullComparisonRightEliminateDistinctEnsureJoinFiltersAndEnsureJoinFiltersPushFilterIntoJoinLeftPushFilterIntoJoinRightPushLimitIntoProjectPushOffsetIntoProjectFoldPlusZeroFoldZeroPlusFoldMinusZeroFoldMultOneFoldOneMultFoldDivOneInvertMinusEliminateUnaryMinusEliminateProjectEliminateProjectProjectFilterUnusedProjectColsFilterUnusedScanColsFilterUnusedSelectColsFilterUnusedLimitColsFilterUnusedOffsetColsFilterUnusedJoinLeftColsFilterUnusedJoinRightColsFilterUnusedAggColsFilterUnusedGroupByColsFilterUnusedValueColsCommuteVarCommuteConstEliminateCoalesceSimplifyCoalesceEliminateCastFoldNullCastFoldNullUnaryFoldNullBinaryLeftFoldNullBinaryRightFoldNullInNonEmptyFoldNullInEmptyFoldNullNotInEmptyNormalizeInConstFoldInNullEnsureSelectFiltersAndEnsureSelectFiltersEliminateSelectMergeSelectsPushSelectIntoProjectPushSelectIntoJoinLeftPushSelectIntoJoinRightMergeSelectInnerJoinPushSelectIntoGroupByPushLimitIntoScanGenerateIndexScansConstrainScanNumRuleNames"

var _RuleName_index = [...]uint16{0, 15, 33, 50, 66, 89, 100, 110, 125, 138, 154, 166, 175, 183, 203, 225, 246, 268, 290, 312, 334, 357, 374, 394, 411, 433, 456, 476, 497, 509, 521, 534, 545, 556, 566, 577, 596, 612, 635, 658, 678, 700, 721, 743, 767, 792, 811, 834, 855, 865, 877, 894, 910, 923, 935, 948, 966, 985, 1003, 1018, 1036, 1052, 1062, 1084, 1103, 1118, 1130, 1151, 1173, 1196, 1216, 1237, 1254, 1272, 1285, 1297}

func (i RuleName) String() string {
	if i >= RuleName(len(_RuleName_index)-1) {
		return "RuleName(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _RuleName_name[_RuleName_index[i]:_RuleName_index[i+1]]
}