package gpt

// Type constants for the GUID for type of partition, see https://en.wikipedia.org/wiki/GUID_Partition_Table#Partition_entries
type Type string

// List of GUID partition types
const (
	Unused                   Type = "00000000-0000-0000-0000-000000000000"
	MbrBoot                  Type = "024DEE41-33E7-11D3-9D69-0008C781F39F"
	EFISystemPartition       Type = "C12A7328-F81F-11D2-BA4B-00A0C93EC93B"
	BiosBoot                 Type = "21686148-6449-6E6F-744E-656564454649"
	MicrosoftReserved        Type = "E3C9E316-0B5C-4DB8-817D-F92DF00215AE"
	MicrosoftBasicData       Type = "EBD0A0A2-B9E5-4433-87C0-68B6B72699C7"
	MicrosoftLDMMetadata     Type = "5808C8AA-7E8F-42E0-85D2-E1E90434CFB3"
	MicrosoftLDMData         Type = "AF9B60A0-1431-4F62-BC68-3311714A69AD"
	MicrosoftWindowsRecovery Type = "DE94BBA4-06D1-4D40-A16A-BFD50179D6AC"
	LinuxFilesystem          Type = "0FC63DAF-8483-4772-8E79-3D69D8477DE4"
	LinuxRaid                Type = "A19D880F-05FC-4D3B-A006-743F0F84911E"
	LinuxRootX86             Type = "44479540-F297-41B2-9AF7-D131D5F0458A"
	LinuxRootX86_64          Type = "4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709"
	LinuxRootArm32           Type = "69DAD710-2CE4-4E3C-B16C-21A1D49ABED3"
	LinuxRootArm64           Type = "B921B045-1DF0-41C3-AF44-4C6F280D3FAE"
	LinuxSwap                Type = "0657FD6D-A4AB-43C4-84E5-0933C84B4F4F"
	LinuxLVM                 Type = "E6D6D379-F507-44C2-A23C-238F2A3DF928"
	LinuxDMCrypt             Type = "7FFEC5C9-2D00-49B7-8941-3EA10A5586B7"
	LinuxLUKS                Type = "CA7D7CCB-63ED-4C53-861C-1742536059CC"
	VMWareFilesystem         Type = "AA31E02A-400F-11DB-9590-000C2911D1B8"
)
