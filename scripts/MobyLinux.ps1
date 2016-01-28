<#
    .SYNOPSIS
        Manages a MobyLinux VM to run Linux Docker on Hyper-V

    .DESCRIPTION
        Creates/Destroys/Starts/Stops A MobyLinux VM to run Docker on Hyper-V

    .PARAMETER IsoFile
        Path to the MobyLinux ISO image, must be set for Create/ReCreate

    .PARAMETER Create
        Create a MobyLinux VM

    .PARAMETER Destroy
        Destroy (remove) a MobyLinux VM

    .PARAMETER ReCreate
        Destroy an existing MobyLinux VM and create a new one

    .PARAMETER Start
        Start an existing MobyLinux VM

    .PARAMETER Stop
        Stop a running MobyLinux VM

    .PARAMETER $VmName
        If passed, use this name for the MobyLinux VM, otherwise 'MobyLinuxVM'

    .PARAMETER $SwitchName
        If passed, use this VMSwitch for network connectivity. Otherwise, use the first existing switch with external connectivity.

    .EXAMPLE
        .\MobyLinux.ps1 -IsoFile .\mobylinux.iso -Create
        .\MobyLinux.ps1 -Start
#>

# This may only work on Windows 10/Windows Server 2016 as we are using a NAT switch

Param(
    [string]
    $VmName = "MobyLinuxVM",

    [string]
    $IsoFile = ".\mobylinux.iso",

    [string]
    $SwitchName,

    [switch]
    $Create,

    [switch]
    $Destroy,

    [switch]
    $Start,

    [switch]
    $Stop
)

$global:VmSwitchName = $SwitchName

# Other hardcoded global parameters
$global:VmMemory = 2147483648  #  2GB
$global:VhdSize = 21474836480  # 20GB
$global:VmProcessors = ([Math]::min((Get-VMHost).LogicalProcessorCount, 2))

# Default location for VHDs
$global:VhdRoot = "$((Get-VMHost).VirtualHardDiskPath)".TrimEnd("\")
# Where we put Moby
$global:VmVhdFile = "$global:VhdRoot\$VmName.vhd"
$global:VmIsoFile = "$global:VhdRoot\$VmName.iso"

# XXX For some reason this works in ISE but not on an elevated Powershell prompt
#function
#Check-Feature
#{
#    [CmdletBinding()]
#    param(
#        [ValidateNotNullOrEmpty()]
#        [string]
#        $FeatureName
#    )
#
#    # WindowsServer and Windows client use differnet commandlets....sigh
#    if (Get-Command Get-WindowsFeature -ErrorAction SilentlyContinue) {
#        if (!(Get-WindowsFeature $FeatureName).Installed) {
#            throw "Please install $FeatureName"
#        }
#    } else {
#        if ((Get-WindowsOptionalFeature -Online -FeatureName $FeatureName).State -eq "Disabled") {
#            throw "Please install $FeatureName"
#        }
#    }
#}

function
Check-Switch
{
    # If no switch name was passed in pick the first external switch
    if ($global:VmSwitchName -eq "") {
        $switches = (Get-VMSwitch |? SwitchType -eq "External")

        if ($switches.Count -gt 0) {
            $global:VmSwitchName = $switches[0].Name
        }
    }

    if ($global:VmSwitchName -ne "") {
        Write-Output "Using external switch: $global:VmSwitchName"
    } else {
        Write-Output "Please create a VMSwitch, e.g."
        Write-Output "New-VMSwitch -Name VirtualSwitch -AllowManagementOS `$True  -NetAdapterName Ethernet0"
        Write-Output "Where Ethernet0 is the name of your main network adapter. See Get-Netadapter"
        throw "No switch"
    }
}


function
Create-MobyLinuxVM
{
    if ($(Get-VM $VmName -ea SilentlyContinue) -ne $null) {
        throw "VM $VmName already exists"
    }

    if (Test-Path $global:VmVhdFile) {
        throw "VHD $global:VmVhdPath already exists"
    }

    if (!(Test-Path $IsoFile)) {
        throw "ISO file at $IsoFile does not exist"
    }

    Write-Output "Creading new dynamic VHD: $global:VmVhdFile"
    $vhd = New-VHD -Path $global:VmVhdFile -SizeBytes $global:VhdSize

    Write-Output "Creating VM $VmName..."
    $vm = New-VM -Name $VmName  -Generation 1 -VHDPath $vhd.Path
    $vm | Set-VM -MemoryStartupBytes $global:VmMemory
    $vm | Set-VMProcessor -Count $global:VmProcessors

    # We use SCSI in the Linux VM
    Add-VMScsiController -VMName $VmName

    # Copy the ISO and add it to the VM
    Copy-Item $IsoFile $global:VmIsoFile
    Add-VMDvdDrive -VMName $VMName -Path $global:VmIsoFile

    # Attach to switch
    $vm | Get-VMNetworkAdapter | Connect-VMNetworkAdapter -SwitchName "$global:VmSwitchName"   

    # Enable Serial Console
    Set-VMComPort -VMName $VmName -number 1 -Path "\\.\pipe\$VmName-com1"
}

function
Destroy-MobyLinuxVM
{
    Write-Output "Destroying $VmName"
    if ($(Get-VM $VmName -ea SilentlyContinue) -ne $null) {
        Remove-VM $VmName -Force
    }

    if (Test-Path $global:VmVhdFile) {
        Remove-Item $global:VmVhdFile
    }

    if (Test-Path $global:VmIsoFile) {
        Remove-Item $global:VmIsoFile
    }
}

function
Start-MobyLinuxVM
{
    Write-Output "Starting $VmName"
    Start-VM -VMName $VmName

    Write-Output 'Connect to Docker by executing:'
    Write-Output '$vmIP=(Get-VMNetworkAdapter MobyLinuxVM).IPAddresses[0]'
    Write-Output '$env:DOCKER_HOST = "tcp://" + $vmIP + ":2375"'
}

function
Stop-MobyLinuxVM
{
    Write-Output "Stopping $VmName"
    # You can use -Force to basically pull the plug on the VM
    # The below requires the Hyper-V tools to be installed in the VM
    Stop-VM -VMName $VmName
}

# Main entry point
# XXX Check if these feature names are the same on Windows Server
# XXX These work when run in ISE but not whe run on normal elevated Powershell
#Check-Feature Microsoft-Hyper-V
#Check-Feature Microsoft-Hyper-V-Hypervisor
#Check-Feature Microsoft-Hyper-V-Management-PowerShell

if ($Stop) {
    Stop-MobyLinuxVM
}
if ($Destroy) {
    Destroy-MobyLinuxVM
}
if ($Create) {
    Check-Switch
    Create-MobyLinuxVM
}
if ($Start) {
    Start-MobyLinuxVM
}
