# Using LinuxKit on Azure

This is a quick guide to running VMs based on LinuxKit images on Azure. Please note that these images currently do not include the [Azure Linux Agent](https://github.com/Azure/WALinuxAgent). As a result, after creating the VM, the portal will report that the creation failed. If you created the VHD properly, you will still be able to SSH into the machine.

When running `linuxkit run azure`, the image you created using `linuxkit build` will be uploaded to Azure in a resource group, and a VM will be created, along with the necessary resources (virtual network, subnet, storage account, network security group, public IP address).

Since Azure does not offer access to the serial output of the VM, you need to have SSH access to the machine in order to attach to it. Please see the example below.


## Setup

You need to authenticate LinuxKit with your Azure subscription. You need to set up the following environment variables:

- `AZURE_TENANT_ID`: The Azure Active Directory tenant ID or domain. You can retrieve this information following [these instructions](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal#get-tenant-id).
- `AZURE_SUBSCRIPTION_ID`: Your Azure Subscription ID. To retrieve it, log in to the Azure portal. The create a [create an Azure Active Directory application and retrieve
its ID and secret following the instructions
here](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-create-service-principal-portal#create-an-azure-active-directory-application).
- `AZURE_CLIENT_ID`: Your Azure Active Directory Application Client ID.
- `AZURE_CLIENT_SECRET`: Your Azure Active Directory Application Secret

[Additional information and required steps for creating a service principal for Azure can be found here](https://docs.docker.com/docker-for-azure/#configuration)


## Build an image

Create a new `azure.yml` file [based on the Azure example](../examples/azure.yml), generate a new SSH key and add it in the `yml`, then `linuxkit build -format vhd azure.yml`.


This will output a `azure.vhd` image.


## Create a new Azure VM based on the image

To deploy the `azure.vhd` image on Azure, invoke the following command:

```
linuxkit run azure -resourceGroupName <resource-group-name> -accountName <storage-account-name> -location westeurope <path-to-your-azure.vhd>
```

Sample output of the command:

```
Creating resource group in westeurope
Creating storage account in westeurope, resource group linuxkit-azure2
2017/05/30 12:51:49 Using default parallelism [8*NumCPU] : 16
Computing MD5 Checksum..
 Completed:  98% RemainingTime: 00h:00m:00s Throughput: 272 MB/sec
Detecting empty ranges..
 Empty ranges : 449/513
Effective upload size: 126.00 MB (from 1024.00 MB originally)
Uploading the VHD..
 Completed: 100% [    126.00 MB] RemainingTime: 00h:00m:00s Throughput: 0 Mb/sec      
Upload completed

OS Image uploaded at https://linuxkitazure2.blob.core.windows.net/linuxkitcontainer/linuxkitimage.vhd
Creating virtual network in resource group linuxkit-azure2, in westeurope
Creating subnet linuxkitsubnet584 in resource group linuxkit-azure2, within virtual network linuxkitvirtualnetwork916
Creating public IP Address in resource group linuxkit-azure2, with name publicip368
publicip368
Started deployment of virtual machine linuxkitvm493 in resource group linuxkit-azure2
Creating virtual machine in resource group linuxkit-azure2, with name linuxkitvm493, in location westeurope
NOTE:  Since you created a minimal VM without the Azure Linux Agent, the portal will notify you that the deployment failed. After around 50 seconds try connecting to the VM

ssh -i path-to-key root@publicip368.westeurope.cloudapp.azure.com

```

After around 50 seconds, try to SSH into the machine (if you added the SSHD service to the image).


## Limitations, workarounds and work in progress

- Since the image currently does not contain the Azure Linux Agent, the Azure Portal will report the creation as failed.
- The main workaround is the way the VHD is uploaded, specifically by using a Docker container based on [Azure VHD Utils](https://github.com/Microsoft/azure-vhd-utils). This is mainly because the tool manages fast and efficient uploads, leveraging parallelism
- There is work in progress to specify what ports to open on the VM (more specifically on a network security group)
- The [metadata package](../pkg/metadata) does not yet support the Azure metadata.
