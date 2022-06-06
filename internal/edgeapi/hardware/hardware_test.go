package hardware_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/project-flotta/flotta-operator/api/v1alpha1"
	"github.com/project-flotta/flotta-operator/internal/edgeapi/hardware"
	"github.com/project-flotta/flotta-operator/models"
)

var (
	testHostname            string   = "the-best-host"
	testPhysicalBytes       int64    = 32000000000
	testUsableBytes         int64    = 30000000000
	testManufacturer        string   = "the-manufacturer"
	testProductName         string   = "the-product"
	testSerialNumber        string   = "0000-1111-2222-3333"
	testVirtual             bool     = true
	testIPv4Addresses       []string = []string{"192.168.85.10", "10.0.23.42"}
	testBootMode            string   = "the-best-boot-mode"
	testPxeInterface        string   = "pxe-interface"
	testCPUCount            int64    = 8
	testArchitecture        string   = "architecture"
	testFlags               []string = []string{"flag1", "flag2"}
	testFrequencyInput      float64  = 3.2
	testFrequencyResult     string   = "3.20"
	testModelName           string   = "xeon"
	testBootable            bool     = true
	testByID                string   = "ByID"
	testByPath              string   = "ByPath"
	testDriveType           string   = "DriveType"
	testHctl                string   = "Hctl"
	testID                  string   = "ID"
	testIoPerf              *models.IoPerf
	testIsInstallationMedia bool                 = true
	testModel               string               = "Model"
	testName                string               = "Name"
	testPath                string               = "Path"
	testSerial              string               = "Serial"
	testSizeBytes           int64                = 200000000000
	testSmart               string               = "Smart"
	testVendor              string               = "Vendor"
	testWwn                 string               = "Wwn"
	testInputFull           *models.HardwareInfo = &models.HardwareInfo{
		Boot: &models.Boot{
			CurrentBootMode: testBootMode,
			PxeInterface:    testPxeInterface,
		},
		CPU: &models.CPU{
			Count:        testCPUCount,
			Architecture: testArchitecture,
			Flags:        testFlags,
			Frequency:    testFrequencyInput,
			ModelName:    testModelName,
		},
		Disks: []*models.Disk{
			{
				Bootable:            testBootable,
				ByID:                testByID,
				ByPath:              testByPath,
				DriveType:           testDriveType,
				Hctl:                testHctl,
				ID:                  testID,
				IoPerf:              testIoPerf,
				IsInstallationMedia: testIsInstallationMedia,
				Model:               testModel,
				Name:                testName,
				Path:                testPath,
				Serial:              testSerial,
				SizeBytes:           testSizeBytes,
				Smart:               testSmart,
				Vendor:              testVendor,
				Wwn:                 testWwn,
			},
			{
				Bootable:            testBootable,
				ByID:                testByID,
				ByPath:              testByPath,
				DriveType:           testDriveType,
				Hctl:                testHctl,
				ID:                  testID,
				IoPerf:              testIoPerf,
				IsInstallationMedia: testIsInstallationMedia,
				Model:               testModel,
				Name:                testName,
				Path:                testPath,
				Serial:              testSerial,
				SizeBytes:           testSizeBytes,
				Smart:               testSmart,
				Vendor:              testVendor,
				Wwn:                 testWwn,
			},
		},
		Gpus: []*models.Gpu{
			{
				Vendor:   "nvidia",
				DeviceID: "aaaaaaa",
			},
			{
				Vendor:   "amd",
				DeviceID: "bbbbbbb",
			},
		},
		Hostname: testHostname,
		Interfaces: []*models.Interface{
			{
				IPV4Addresses: testIPv4Addresses,
				Name:          "eth0",
			},
			{
				IPV4Addresses: testIPv4Addresses,
				Name:          "eth1",
			},
		},
		Memory: &models.Memory{
			PhysicalBytes: testPhysicalBytes,
			UsableBytes:   testUsableBytes,
		},
		SystemVendor: &models.SystemVendor{
			Manufacturer: testManufacturer,
			ProductName:  testProductName,
			SerialNumber: testSerialNumber,
			Virtual:      testVirtual,
		},
		HostDevices: []*models.HostDevice{
			{
				Path:       "/dev/loop1",
				DeviceType: "block",
				UID:        1,
				Gid:        1,
				Major:      1,
				Minor:      1,
			},
			{
				Path:       "/dev/loop2",
				DeviceType: "char",
				UID:        2,
				Gid:        2,
				Major:      2,
				Minor:      2,
			},
		},
	}
)

var _ = Describe("Hardware", func() {
	Context("MapHardware", func() {
		It("should accept nil input", func() {
			// given
			var input *models.HardwareInfo

			// when
			result := hardware.MapHardware(input)

			// then
			Expect(result).To(BeNil())
		})

		It("should handle nil fields in input", func() {
			// given
			input := models.HardwareInfo{}

			// when
			result := hardware.MapHardware(&input)

			// then
			Expect(result).NotTo(BeNil())
		})

		It("should map all fields", func() {
			// given
			input := testInputFull

			//when
			result := hardware.MapHardware(input)

			// then
			Expect(result).NotTo(BeNil())
			Expect(result.Boot).NotTo(BeNil())
			Expect(result.CPU).NotTo(BeNil())
			Expect(result.Disks).NotTo(BeNil())
			Expect(result.Disks).To(HaveLen(len(input.Disks)))
			Expect(result.Gpus).NotTo(BeNil())
			Expect(result.Gpus).To(HaveLen(len(input.Gpus)))
			Expect(result.Hostname).To(Equal(input.Hostname))
			Expect(result.Interfaces).NotTo(BeNil())
			Expect(result.Interfaces).To(HaveLen(len(input.Interfaces)))
			Expect(result.Memory).NotTo(BeNil())
			Expect(result.SystemVendor).NotTo(BeNil())

			Expect(result.Boot.CurrentBootMode).To(Equal(testBootMode))
			Expect(result.Boot.PxeInterface).To(Equal(testPxeInterface))
			Expect(result.CPU.Count).To(Equal(testCPUCount))
			Expect(result.CPU.Architecture).To(Equal(testArchitecture))
			Expect(result.CPU.Flags).To(Equal(testFlags))
			Expect(result.CPU.Frequency).To(Equal(testFrequencyResult))
			Expect(result.CPU.ModelName).To(Equal(testModelName))

			for i := range input.Disks {
				dV1 := result.Disks[i]
				Expect(dV1.Bootable).To(Equal(testBootable))
				Expect(dV1.ByID).To(Equal(testByID))
				Expect(dV1.ByPath).To(Equal(testByPath))
				Expect(dV1.DriveType).To(Equal(testDriveType))
				Expect(dV1.Hctl).To(Equal(testHctl))
				Expect(dV1.ID).To(Equal(testID))
				Expect(dV1.IoPerf).To(Equal((*v1alpha1.IoPerf)(testIoPerf)))
				Expect(dV1.IsInstallationMedia).To(Equal(testIsInstallationMedia))
				Expect(dV1.Model).To(Equal(testModel))
				Expect(dV1.Name).To(Equal(testName))
				Expect(dV1.Path).To(Equal(testPath))
				Expect(dV1.Serial).To(Equal(testSerial))
				Expect(dV1.SizeBytes).To(Equal(testSizeBytes))
				Expect(dV1.Smart).To(Equal(testSmart))
				Expect(dV1.Vendor).To(Equal(testVendor))
				Expect(dV1.Wwn).To(Equal(testWwn))
			}
			for i, g := range input.Gpus {
				gV1 := result.Gpus[i]
				Expect(gV1).To(Equal((*v1alpha1.Gpu)(g)))
			}
			for i, inter := range input.Interfaces {
				interV1 := result.Interfaces[i]
				Expect(interV1).To(Equal((*v1alpha1.Interface)(inter)))
			}
			for i, dev := range input.HostDevices {
				devV1 := result.HostDevices[i]
				Expect(devV1.Path).To(Equal(dev.Path))
				Expect(devV1.DeviceType).To(Equal(dev.DeviceType))
				Expect(devV1.GID).To(Equal(uint32(dev.Gid)))
				Expect(devV1.UID).To(Equal(uint32(dev.UID)))
				Expect(devV1.Major).To(Equal(uint32(dev.Major)))
				Expect(devV1.Minor).To(Equal(uint32(dev.Minor)))
			}
			Expect(result.Memory.PhysicalBytes).To(Equal(testPhysicalBytes))
			Expect(result.Memory.UsableBytes).To(Equal(testUsableBytes))
			Expect(result.SystemVendor.Manufacturer).To(Equal(testManufacturer))
			Expect(result.SystemVendor.ProductName).To(Equal(testProductName))
			Expect(result.SystemVendor.SerialNumber).To(Equal(testSerialNumber))
			Expect(result.SystemVendor.Virtual).To(Equal(testVirtual))
		})
	})

	Context("MapLabels", func() {
		It("should accept nil input", func() {
			// given
			var input *models.HardwareInfo

			// when
			result := hardware.MapLabels(input)

			// then
			Expect(result).To(BeNil())
		})

		It("should handle nil fields in input", func() {
			// given
			input := models.HardwareInfo{}

			// when
			result := hardware.MapHardware(&input)

			// then
			Expect(result).NotTo(BeNil())
		})

		It("should map all labels", func() {
			// given
			input := testInputFull

			//when
			result := hardware.MapLabels(input)

			// then
			Expect(result).NotTo(BeNil())
			Expect(len(result)).To(Equal(6))
			Expect(result["device.hostname"]).To(Equal(testHostname))
			Expect(result["device.cpu-architecture"]).To(Equal(strings.ToLower(testArchitecture)))
			Expect(result["device.cpu-model"]).To(Equal(strings.ToLower(testModelName)))
			Expect(result["device.system-manufacturer"]).To(Equal(strings.ToLower(testManufacturer)))
			Expect(result["device.system-product"]).To(Equal(strings.ToLower(testProductName)))
			Expect(result["device.system-serial"]).To(Equal(testSerialNumber))
		})
	})
})
