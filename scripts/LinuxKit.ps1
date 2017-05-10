<#
    .SYNOPSIS
        Manages a LinuxKit VM on Hyper-V

    .DESCRIPTION
        Creates/Destroys/Starts/Stops a LinuxKit VM.
        It creates a Gen2 VM which boots EFI ISOs

    .PARAMETER VmName
        If passed, use this name for the LinuxKit VM, otherwise 'LinuxKitVM'

    .PARAMETER IsoFile
        Path to the ISO image to boot from, must be set for Create/ReCreate

    .PARAMETER SwitchName
        Use this switch to connect the VM to

    .PARAMETER Create
        Create a LinuxKit VM

    .PARAMETER Memory
        Memory allocated for the VM in MB (optional on Create, default: 1024 MB)

    .PARAMETER CPUs
        Number of CPUs to use (optional on Create, default: 1)

    .PARAMETER DiskSize
        Disk size in MB (optional on Create, default: 0, no disk)

    .PARAMETER Destroy
        Remove a LinuxKit VM

    .PARAMETER Start
        Start an existing LinuxKit VM

    .PARAMETER Stop
        Stop a running LinuxKit VM

    .EXAMPLE
        .\LinuxKit.ps1 -IsoFile .\linuxkit-efi.iso -Create
        .\LinuxKit.ps1 -Start
#>

Param(
    [string] $VmName = "LinuxKitVM",
    [string] $IsoFile = ".\linuxkit-efi.iso",
    [string] $SwicthName = ".\linuxkit-efi.iso",

    [Parameter(ParameterSetName='Create',Mandatory=$false)][switch] $Create,
    [Parameter(ParameterSetName='Create',Mandatory=$false)][int] $CPUs = 1,
    [Parameter(ParameterSetName='Create',Mandatory=$false)][long] $Memory = 1024,
    [Parameter(ParameterSetName='Create',Mandatory=$false)][long] $DiskSize = 0,
    [Parameter(ParameterSetName='Create',Mandatory=$false)][string] $SwitchName = "",
    [Parameter(ParameterSetName='Remove',Mandatory=$false)][switch] $Remove,
    [Parameter(ParameterSetName='Start',Mandatory=$false)][switch] $Start,
    [Parameter(ParameterSetName='Stop',Mandatory=$false)][switch] $Stop
)

# Make sure we stop at Errors unless otherwise explicitly specified
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Explicitly disable Module autoloading and explicitly import the
# Modules this script relies on. This is not strictly necessary but
# good practise as it prevents arbitrary errors
$PSModuleAutoloadingPreference = 'None'

Import-Module Microsoft.PowerShell.Utility
Import-Module Microsoft.PowerShell.Management
Import-Module Hyper-V

function Get-Vhd-Root {
    if($VhdPathOverride){
        return $VhdPathOverride
    }
    # Default location for VHDs
    $VhdRoot = "$((Get-VMHost).VirtualHardDiskPath)".TrimEnd("\")
    return "$VhdRoot\$VmName.vhdx"
}

# Posh thinks that Create is an unapproved verb
function New-LinuxKitVM {
    if (!(Test-Path $IsoFile)) {
        Fatal "ISO file at $IsoFile does not exist"
    }

    $CPUs = [Math]::min((Get-VMHost).LogicalProcessorCount, $CPUs)

    $vm = Get-VM $VmName -ea SilentlyContinue
    if ($vm) {
        if ($vm.Length -ne 1) {
            Fatal "Multiple VMs exist with the name $VmName. Delete invalid ones."
        }
    } else {
        Write-Output "Creating VM $VmName..."
        $vm = New-VM -Name $VmName -Generation 2 -NoVHD
        $vm | Set-VM -AutomaticStartAction Nothing -AutomaticStopAction ShutDown -CheckpointType Disabled
    }

    if ($vm.Generation -ne 2) {
        Fatal "VM $VmName is a Generation $($vm.Generation) VM. It should be a Generation 2."
    }

    if ($vm.State -ne "Off") {
        Write-Output "VM $VmName is $($vm.State). Cannot change its settings."
        return
    }

    Write-Output "Setting CPUs to $CPUs and Memory to $Memory MB"
    $Memory = ([Math]::min($Memory, ($vm | Get-VMMemory).MaximumPerNumaNode))
    $vm | Set-VM -MemoryStartupBytes ($Memory*1024*1024) -ProcessorCount $CPUs -StaticMemory

    if ($DiskSize -ne 0) {
        $VmVhdFile = Get-Vhd-Root
        $vhd = Get-VHD -Path $VmVhdFile -ea SilentlyContinue
        if (!$vhd) {
            Write-Output "Creating dynamic VHD: $VmVhdFile"
            $vhd = New-VHD -Path $VmVhdFile -Dynamic -SizeBytes ($DiskSize*1024*1024)
        }
        Write-Output "Attach VHD $VmVhdFile"
        $vm | Add-VMHardDiskDrive -Path $VmVhdFile
    }

    if ($SwitchName -ne "") {
        $vmNetAdapter = $vm | Get-VMNetworkAdapter
        if (!$vmNetAdapter) {
            Write-Output "Attach Net Adapter"
            $vmNetAdapter = $vm | Add-VMNetworkAdapter -Passthru
        }

        Write-Output "Connect to switch $SwicthName"
        $vmNetAdapter | Connect-VMNetworkAdapter -VMSwitch $(Get-VMSwitch $SwitchName)
    }

    Write-Output "Attach DVD $IsoFile"
    $vm | Add-VMDvdDrive -Path $IsoFile

    $iso = $vm | Get-VMFirmware | select -ExpandProperty BootOrder | ? { $_.FirmwarePath.EndsWith("Scsi(0,1)") }
    $vm | Set-VMFirmware -EnableSecureBoot Off -FirstBootDevice $iso
    $vm | Set-VMComPort -number 1 -Path "\\.\pipe\$VmName-com1"

    Write-Output "VM created."
}

function Remove-LinuxKitVM {
    Write-Output "Removing VM $VmName..."

    Remove-VM $VmName -Force -ea SilentlyContinue
}

function Start-LinuxKitVM {
    Write-Output "Starting VM $VmName..."
    Start-VM -VMName $VmName
}

function Stop-LinuxKitVM {
    $vm = Get-VM $VmName -ea SilentlyContinue
    if (!$vm) {
        Write-Output "VM $VmName does not exist"
        return
    }

    # This is a bit harsh, we poweroff
    $vm | Stop-VM -Confirm:$false -TurnOff -Force -ea SilentlyContinue
}


function Fatal {
    throw "$args"
    Exit 1
}

# Main entry point
Try {
    Switch ($PSBoundParameters.GetEnumerator().Where({$_.Value -eq $true}).Key) {
        'Create'   { New-LinuxKitVM }
        'Start'    { Start-LinuxKitVM }
        'Stop'     { Stop-LinuxKitVM }
        'Remove'   { Stop-LinuxKitVM; Remove-LinuxKitVM }
    }
} Catch {
    throw
    Exit 1
}
