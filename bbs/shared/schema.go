package shared

import (
	"path"
	"strconv"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

const SchemaRoot = "/v1/"
const CellSchemaRoot = SchemaRoot + "cell"
const ReceptorSchemaRoot = SchemaRoot + "receptor"
const ActualLRPSchemaRoot = SchemaRoot + "actual"
const DesiredLRPSchemaRoot = SchemaRoot + "desired"
const VolumeSetSchemaRoot = SchemaRoot + "volume_set"
const VolumeSchemaRoot = SchemaRoot + "volume"
const TaskSchemaRoot = SchemaRoot + "task"
const LockSchemaRoot = SchemaRoot + "locks"
const DomainSchemaRoot = SchemaRoot + "domain"

const ActualLRPInstanceKey = "instance"
const ActualLRPEvacuatingKey = "evacuating"

func CellSchemaPath(cellID string) string {
	return path.Join(CellSchemaRoot, cellID)
}

func ReceptorSchemaPath(receptorID string) string {
	return path.Join(ReceptorSchemaRoot, receptorID)
}

func ActualLRPProcessDir(processGuid string) string {
	return path.Join(ActualLRPSchemaRoot, processGuid)
}

func ActualLRPIndexDir(processGuid string, index int) string {
	return path.Join(ActualLRPProcessDir(processGuid), strconv.Itoa(index))
}

func ActualLRPSchemaPath(processGuid string, index int) string {
	return path.Join(ActualLRPIndexDir(processGuid, index), ActualLRPInstanceKey)
}

func EvacuatingActualLRPSchemaPath(processGuid string, index int) string {
	return path.Join(ActualLRPIndexDir(processGuid, index), ActualLRPEvacuatingKey)
}

func DesiredLRPSchemaPath(lrp models.DesiredLRP) string {
	return DesiredLRPSchemaPathByProcessGuid(lrp.ProcessGuid)
}

func DesiredLRPSchemaPathByProcessGuid(processGuid string) string {
	return path.Join(DesiredLRPSchemaRoot, processGuid)
}

func VolumeSetSchemaPath(volumeSetGuid string) string {
	return path.Join(VolumeSetSchemaRoot, volumeSetGuid)
}

func VolumeDir(volumeSetGuid string) string {
	return path.Join(VolumeSchemaRoot, volumeSetGuid)
}

func VolumeSchemaPath(volumeSetGuid string, index int) string {
	return path.Join(VolumeDir(volumeSetGuid), strconv.Itoa(index))
}

func TaskSchemaPath(taskGuid string) string {
	return path.Join(TaskSchemaRoot, taskGuid)
}

func LockSchemaPath(lockName string) string {
	return path.Join(LockSchemaRoot, lockName)
}

func DomainSchemaPath(domain string) string {
	return path.Join(DomainSchemaRoot, domain)
}
