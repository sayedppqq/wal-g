package greenplum_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wal-g/wal-g/internal"
	"github.com/wal-g/wal-g/internal/databases/greenplum"
	"github.com/wal-g/wal-g/testtools"
)

var (
	testStreamBackup = internal.BackupTime{
		BackupName: "stream_20210329T125616Z",
		Time:       time.Now(),
	}
)

func TestGetRestorePointTimeSlices_emptyList(t *testing.T) {
	folder := testtools.MakeDefaultInMemoryStorageFolder()
	_ = folder.PutObject("restore_123312", &bytes.Buffer{})
	objects, _, _ := folder.ListFolder()
	result := greenplum.GetRestorePointsTimeSlices(objects)
	assert.Equalf(t, []greenplum.RestorePointTime{}, result, "GetRestorePointsTimeSlices returned not empty list: something wrong")
}

func TestGetRestorePointTimeSlices_List(t *testing.T) {
	folder := testtools.MakeDefaultInMemoryStorageFolder()
	_ = folder.PutObject("restore_123312", &bytes.Buffer{})
	_ = folder.PutObject(testStreamBackup.BackupName+greenplum.RestorePointSuffix, &bytes.Buffer{})

	objects, _, _ := folder.ListFolder()

	result := greenplum.GetRestorePointsTimeSlices(objects)

	assert.Equalf(t, 1, len(result), "GetRestorePointTimeSlices returned wrong count of backup: something wrong")
	assert.Equalf(t, testStreamBackup.BackupName, result[0].Name, "GetRestorePointTimeSlices returned strange name")
	assert.True(t, testStreamBackup.Time.Before(result[0].Time), "GetRestorePointTimeSlices returned bad time: storage time less than mock time")
}

func TestGetRestorePointTimeSlices_OrderCheck(t *testing.T) {
	folder := testtools.MakeDefaultInMemoryStorageFolder()
	_ = folder.PutObject(testStreamBackup.BackupName+".1"+greenplum.RestorePointSuffix, &bytes.Buffer{})
	time.Sleep(time.Second)
	_ = folder.PutObject(testStreamBackup.BackupName+".2"+greenplum.RestorePointSuffix, &bytes.Buffer{})

	objects, _, _ := folder.ListFolder()

	result := greenplum.GetRestorePointsTimeSlices(objects)

	assert.Equalf(t, 2, len(result), "GetRestorePointTimeSlices returned wrong count of restore points: something wrong")
	assert.True(t, result[0].Name == testStreamBackup.BackupName+".1", "GetRestorePointTimeSlices returned bad time ordering: "+testStreamBackup.BackupName+".1 should be first, because second was added earlier")
	assert.True(t, result[0].Time.Before(result[1].Time), "GetRestorePointTimeSlices returned bad time ordering: order should be Ascending")
}

func TestFindRestorePointBeforeTS_Correct(t *testing.T) {
	targetTs := time.Now()
	restorePoints := []greenplum.RestorePointMetadata{
		{
			Name:       "too_old_restore_point",
			StartTime:  targetTs.Add(-1 * time.Hour),
			FinishTime: targetTs.Add(-1 * time.Hour),
		}, {
			Name:       "too_new_restore_point",
			StartTime:  targetTs.Add(1 * time.Hour),
			FinishTime: targetTs.Add(1 * time.Hour),
		}, {
			Name:       "expected_restore_point",
			StartTime:  targetTs.Add(-11 * time.Second),
			FinishTime: targetTs.Add(-10 * time.Second),
		}}

	found, err := greenplum.FindRestorePointBeforeTS(targetTs.Format(time.RFC3339), restorePoints)
	assert.NoError(t, err)
	assert.Equal(t, "expected_restore_point", found)
}

func TestFindRestorePointWithTS_StartInPast(t *testing.T) {
	targetTs := time.Now()
	restorePoints := []greenplum.RestorePointMetadata{
		{
			Name:       "too_old_restore_point",
			StartTime:  targetTs.Add(-1 * time.Hour),
			FinishTime: targetTs.Add(-1 * time.Hour),
		},
		{
			Name:       "too_old_restore_point2",
			StartTime:  targetTs.Add(-11 * time.Second),
			FinishTime: targetTs.Add(-1 * time.Second),
		},
		{
			Name:       "too_new_restore_point",
			StartTime:  targetTs.Add(1 * time.Hour),
			FinishTime: targetTs.Add(1 * time.Hour),
		}, {
			Name:       "target_restore_point",
			StartTime:  targetTs.Add(-11 * time.Second),
			FinishTime: targetTs.Add(time.Second),
		}}

	found, err := greenplum.FindRestorePointWithTS(targetTs.Format(time.RFC3339), restorePoints)
	assert.NoError(t, err)
	assert.Equal(t, "target_restore_point", found)
}

func TestFindRestorePointWithTS_StartInTS(t *testing.T) {
	targetTs := time.Now()
	restorePoints := []greenplum.RestorePointMetadata{
		{
			Name:       "too_old_restore_point",
			StartTime:  targetTs.Add(-1 * time.Hour),
			FinishTime: targetTs,
		},
		{
			Name:       "too_old_restore_point2",
			StartTime:  targetTs.Add(-11 * time.Second),
			FinishTime: targetTs.Add(-1 * time.Second),
		},
		{
			Name:       "too_new_restore_point",
			StartTime:  targetTs,
			FinishTime: targetTs.Add(1 * time.Hour),
		}, {
			Name:       "target_restore_point",
			StartTime:  targetTs,
			FinishTime: targetTs.Add(1 * time.Second),
		}}

	found, err := greenplum.FindRestorePointWithTS(targetTs.Format(time.RFC3339), restorePoints)
	assert.NoError(t, err)
	assert.Equal(t, "target_restore_point", found)
}

func TestFindRestorePointWithTS_StartInFuture(t *testing.T) {
	targetTs := time.Now()
	restorePoints := []greenplum.RestorePointMetadata{
		{
			Name:       "too_old_restore_point",
			StartTime:  targetTs.Add(-1 * time.Hour),
			FinishTime: targetTs.Add(-1 * time.Hour),
		},
		{
			Name:       "too_new_restore_point",
			StartTime:  targetTs.Add(1 * time.Hour),
			FinishTime: targetTs.Add(1 * time.Hour),
		}, {
			Name:       "target_restore_point",
			StartTime:  targetTs.Add(11 * time.Second),
			FinishTime: targetTs.Add(12 * time.Second),
		}}

	found, err := greenplum.FindRestorePointWithTS(targetTs.Format(time.RFC3339), restorePoints)
	assert.NoError(t, err)
	assert.Equal(t, "target_restore_point", found)
}

func TestFindRestorePointBeforeTS_NoRestorePoints(t *testing.T) {
	targetTs := time.Now()

	found, err := greenplum.FindRestorePointBeforeTS(targetTs.Format(time.RFC3339), []greenplum.RestorePointMetadata{})
	assert.Error(t, err)
	assert.IsType(t, greenplum.NoRestorePointsFoundError{}, err)
	assert.Equal(t, "", found)
}

func TestFindRestorePointBeforeTS_NoMatches(t *testing.T) {
	targetTs := time.Now()
	restorePoints := []greenplum.RestorePointMetadata{
		{
			Name:       "too_new_restore_point1",
			StartTime:  targetTs.Add(2 * time.Hour),
			FinishTime: targetTs.Add(2 * time.Hour),
		}, {
			Name:       "too_new_restore_point2",
			StartTime:  targetTs.Add(1 * time.Hour),
			FinishTime: targetTs.Add(1 * time.Hour),
		}, {
			Name:       "too_new_restore_point3",
			StartTime:  targetTs.Add(1 * time.Second),
			FinishTime: targetTs.Add(1 * time.Second),
		}}

	found, err := greenplum.FindRestorePointBeforeTS(targetTs.Format(time.RFC3339), restorePoints)
	assert.Error(t, err)
	assert.IsType(t, greenplum.NoRestorePointsFoundError{}, err)
	assert.Equal(t, "", found)
}

func TestFindRestorePointBeforeTS_ExactTime(t *testing.T) {
	targetStr := "2022-12-22T14:00:02.37584Z"
	targetTs, _ := time.Parse(time.RFC3339, targetStr)
	restorePoints := []greenplum.RestorePointMetadata{
		{
			Name:       "too_old_restore_point",
			StartTime:  targetTs.Add(-1 * time.Nanosecond),
			FinishTime: targetTs.Add(-1 * time.Nanosecond),
		}, {
			Name:       "expected_restore_point",
			StartTime:  targetTs,
			FinishTime: targetTs,
		}, {
			Name:       "too_new_restore_point",
			StartTime:  targetTs.Add(1 * time.Nanosecond),
			FinishTime: targetTs.Add(1 * time.Nanosecond),
		}}

	found, err := greenplum.FindRestorePointBeforeTS(targetStr, restorePoints)
	assert.NoError(t, err)
	assert.Equal(t, "expected_restore_point", found)
}
