// Code generated by "stringer"; DO NOT EDIT.

package screl

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[DescID-1]
	_ = x[IndexID-2]
	_ = x[ColumnFamilyID-3]
	_ = x[ColumnID-4]
	_ = x[ConstraintID-5]
	_ = x[Name-6]
	_ = x[ReferencedDescID-7]
	_ = x[Comment-8]
	_ = x[TemporaryIndexID-9]
	_ = x[SourceIndexID-10]
	_ = x[TargetStatus-11]
	_ = x[CurrentStatus-12]
	_ = x[Element-13]
	_ = x[Target-14]
	_ = x[ReferencedTypeIDs-15]
	_ = x[ReferencedSequenceIDs-16]
	_ = x[ReferencedFunctionIDs-17]
	_ = x[AttrMax-17]
}

func (i Attr) String() string {
	switch i {
	case DescID:
		return "DescID"
	case IndexID:
		return "IndexID"
	case ColumnFamilyID:
		return "ColumnFamilyID"
	case ColumnID:
		return "ColumnID"
	case ConstraintID:
		return "ConstraintID"
	case Name:
		return "Name"
	case ReferencedDescID:
		return "ReferencedDescID"
	case Comment:
		return "Comment"
	case TemporaryIndexID:
		return "TemporaryIndexID"
	case SourceIndexID:
		return "SourceIndexID"
	case TargetStatus:
		return "TargetStatus"
	case CurrentStatus:
		return "CurrentStatus"
	case Element:
		return "Element"
	case Target:
		return "Target"
	case ReferencedTypeIDs:
		return "ReferencedTypeIDs"
	case ReferencedSequenceIDs:
		return "ReferencedSequenceIDs"
	case ReferencedFunctionIDs:
		return "ReferencedFunctionIDs"
	default:
		return "Attr(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
