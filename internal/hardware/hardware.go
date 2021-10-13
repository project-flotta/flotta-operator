package hardware

import (
	"fmt"

	"github.com/jakub-dzon/k4e-operator/api/v1alpha1"
	"github.com/jakub-dzon/k4e-operator/models"
)

func MapHardware(hardware *models.HardwareInfo) *v1alpha1.Hardware {
	if hardware == nil {
		return nil
	}

	disks := []*v1alpha1.Disk{}
	for _, d := range hardware.Disks {
		diskV1 := v1alpha1.Disk{
			Bootable:            d.Bootable,
			ByID:                d.ByID,
			ByPath:              d.ByPath,
			DriveType:           d.DriveType,
			Hctl:                d.Hctl,
			ID:                  d.ID,
			IoPerf:              (*v1alpha1.IoPerf)(d.IoPerf),
			IsInstallationMedia: d.IsInstallationMedia,
			Model:               d.Model,
			Name:                d.Name,
			Path:                d.Path,
			Serial:              d.Serial,
			SizeBytes:           d.SizeBytes,
			Smart:               d.Smart,
			Vendor:              d.Vendor,
			Wwn:                 d.Wwn,
		}
		disks = append(disks, &diskV1)
	}

	gpus := []*v1alpha1.Gpu{}
	for _, g := range hardware.Gpus {
		gpus = append(gpus, (*v1alpha1.Gpu)(g))
	}

	interfaces := []*v1alpha1.Interface{}
	for _, i := range hardware.Interfaces {
		interfaces = append(interfaces, (*v1alpha1.Interface)(i))
	}

	hw := v1alpha1.Hardware{
		Hostname: hardware.Hostname,

		Gpus:       gpus,
		Disks:      disks,
		Interfaces: interfaces,
	}
	if hardware.Boot != nil {
		hw.Boot = &v1alpha1.Boot{
			CurrentBootMode: hardware.Boot.CurrentBootMode,
			PxeInterface:    hardware.Boot.PxeInterface,
		}
	}

	cpu := hardware.CPU
	if cpu != nil {
		hw.CPU = &v1alpha1.CPU{
			Architecture: cpu.Architecture,
			Count:        cpu.Count,
			Flags:        cpu.Flags,
			Frequency:    fmt.Sprintf("%.2f", cpu.Frequency),
			ModelName:    cpu.ModelName,
		}
	}

	memory := hardware.Memory
	if memory != nil {
		hw.Memory = &v1alpha1.Memory{
			PhysicalBytes: memory.PhysicalBytes,
			UsableBytes:   memory.UsableBytes,
		}
	}

	systemVendor := hardware.SystemVendor
	if systemVendor != nil {
		hw.SystemVendor = &v1alpha1.SystemVendor{
			Manufacturer: systemVendor.Manufacturer,
			ProductName:  systemVendor.ProductName,
			SerialNumber: systemVendor.SerialNumber,
			Virtual:      systemVendor.Virtual,
		}
	}
	return &hw
}
